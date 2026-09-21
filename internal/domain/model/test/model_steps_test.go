package model_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"io/fs"
	"strconv"
	"strings"
	"testing/fstest"
)

// countingFS wraps fstest.MapFS, counting Open calls per path so a test can
// prove a lazy loader reads a file at most once even across repeat calls.
type countingFS struct {
	files  fstest.MapFS
	counts map[string]int
}

func (f countingFS) Open(name string) (fs.File, error) {
	f.counts[name]++
	return f.files.Open(name)
}

func initialize(sc *godog.ScenarioContext) {
	var ownRoot, ownMain string
	var ownFormat model.SkillFormat
	sc.Step(`^a skill rooted at "([^"]*)" main "([^"]*)" flat "([^"]*)"$`, func(ctx context.Context, root, main, flat string) error {
		ownRoot, ownMain = root, main
		ownFormat = model.AgentDirSkill
		if flat == "true" {
			ownFormat = model.FlatSkill
		}
		return nil
	})
	sc.Step(`^path "([^"]*)" is owned "([^"]*)" and relative is "([^"]*)"$`, func(ctx context.Context, p, owned, relative string) error {
		gotOwned := model.OwnsPath(ownMain, ownRoot, ownFormat, p)
		gotRelative := model.RelativePath(ownMain, ownRoot, p)
		return testsupport.Equal([]any{gotOwned, gotRelative}, []any{owned == "true", relative})
	})

	var lazySkill *model.Skill
	var lazyIndex map[string]int
	var lazyCounts map[string]int
	var lazyData []byte
	var lazyErr error
	sc.Step(`^a skill "([^"]*)" rooted at "([^"]*)" main "([^"]*)" with nested file "([^"]*)" containing "([^"]*)"$`, func(ctx context.Context, name, root, main, path, content string) error {
		lazyCounts = map[string]int{}
		files := fstest.MapFS{model.NestedRepoPath(root, path): &fstest.MapFile{Data: []byte(content)}}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: countingFS{files: files, counts: lazyCounts}}
		lazySkill = &model.Skill{Name: name, Main: main, Root: root, Format: model.AgentDirSkill, Repo: repo, Files: []model.File{{Path: path}}}
		lazyIndex = map[string]int{path: 0}
		lazyData, lazyErr = nil, nil
		return nil
	})
	sc.Step(`^a skill "([^"]*)" rooted at "([^"]*)" main "([^"]*)" with a missing nested file "([^"]*)"$`, func(ctx context.Context, name, root, main, path string) error {
		lazyCounts = map[string]int{}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: countingFS{files: fstest.MapFS{}, counts: lazyCounts}}
		lazySkill = &model.Skill{Name: name, Main: main, Root: root, Format: model.AgentDirSkill, Repo: repo, Files: []model.File{{Path: path}}}
		lazyIndex = map[string]int{path: 0}
		lazyData, lazyErr = nil, nil
		return nil
	})
	sc.Step(`^nested file "([^"]*)" data is not yet loaded$`, func(ctx context.Context, path string) error {
		if lazySkill.Files[lazyIndex[path]].Data != nil {
			return fmt.Errorf("expected %q Data to be nil before FileData runs", path)
		}
		return nil
	})
	sc.Step(`^I read nested file "([^"]*)" data(?: again)?$`, func(ctx context.Context, path string) error {
		lazyData, lazyErr = lazySkill.FileData(lazyIndex[path])
		return nil
	})
	sc.Step(`^nested file "([^"]*)" data is "([^"]*)"$`, func(ctx context.Context, path, want string) error {
		if lazyErr != nil {
			return lazyErr
		}
		return testsupport.Equal(string(lazyData), want)
	})
	sc.Step(`^nested file "([^"]*)" was read "([^"]*)" times?$`, func(ctx context.Context, path, want string) error {
		return testsupport.Equal(strconv.Itoa(lazyCounts[model.NestedRepoPath(lazySkill.Root, path)]), want)
	})
	sc.Step(`^reading nested file data fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if lazyErr == nil || !strings.Contains(lazyErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", lazyErr, want)
		}
		return nil
	})

	var catalog *model.SkillCatalog
	var catalogErr error
	sc.Step(`^a catalog with conflict policy "([^"]*)"$`, func(ctx context.Context, policy string) error {
		catalog = &model.SkillCatalog{Conflict: policy}
		catalogErr = nil
		return nil
	})
	sc.Step(`^I add skill "([^"]*)" from repo "([^"]*)" main "([^"]*)"$`, func(ctx context.Context, name, repo, main string) error {
		skill := &model.Skill{Name: name, Main: main, Root: main[:strings.LastIndex(main, "/")], Format: model.AgentDirSkill, Repo: &model.Repository{ID: repo}}
		catalogErr = catalog.GetOrAdd(ctx, skill)
		return nil
	})
	sc.Step(`^I add skill "([^"]*)" from repo "([^"]*)" main "([^"]*)" with files "([^"]*)"$`, func(ctx context.Context, name, repo, main, filesArg string) error {
		var files []model.File
		for _, p := range strings.Split(filesArg, ",") {
			files = append(files, model.File{Path: p})
		}
		skill := &model.Skill{Name: name, Main: main, Root: main[:strings.LastIndex(main, "/")], Format: model.AgentDirSkill, Repo: &model.Repository{ID: repo}, Files: files}
		catalogErr = catalog.GetOrAdd(ctx, skill)
		return nil
	})
	sc.Step(`^catalog destination of "([^"]*)" path "([^"]*)" is "([^"]*)" at "([^"]*)"$`, func(ctx context.Context, repo, p, name, dest string) error {
		gotName, gotDest, ok := catalog.Destination(repo, p)
		if !ok {
			return fmt.Errorf("destination not found for %s %s", repo, p)
		}
		return testsupport.Equal([]any{gotName, gotDest}, []any{name, dest})
	})
	sc.Step(`^catalog destination of "([^"]*)" path "([^"]*)" is not found$`, func(ctx context.Context, repo, p string) error {
		if _, _, ok := catalog.Destination(repo, p); ok {
			return fmt.Errorf("expected no destination for %s %s", repo, p)
		}
		return nil
	})
	sc.Step(`^catalog destinations map has "([^"]*)" "([^"]*)" pointing to "([^"]*)"$`, func(ctx context.Context, repo, p, dest string) error {
		all := catalog.Destinations()
		got, ok := all[model.OriginalKey(repo, p)]
		if !ok {
			return fmt.Errorf("destinations map missing entry for %s %s", repo, p)
		}
		return testsupport.Equal(got, dest)
	})
	sc.Step(`^catalog destinations map has no entry for "([^"]*)" "([^"]*)"$`, func(ctx context.Context, repo, p string) error {
		if _, ok := catalog.Destinations()[model.OriginalKey(repo, p)]; ok {
			return fmt.Errorf("expected no destinations map entry for %s %s", repo, p)
		}
		return nil
	})
	sc.Step(`^catalog error contains "([^"]*)"$`, func(ctx context.Context, want string) error {
		if want == "" {
			return catalogErr
		}
		if catalogErr == nil || !strings.Contains(catalogErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", catalogErr, want)
		}
		return nil
	})
	sc.Step(`^catalog skills are "([^"]*)"$`, func(ctx context.Context, want string) error {
		var names []string
		for _, s := range catalog.Skills {
			names = append(names, s.Name)
		}
		return testsupport.Equal(strings.Join(names, ","), want)
	})
	sc.Step(`^catalog skill "([^"]*)" belongs to repo "([^"]*)"$`, func(ctx context.Context, name, repo string) error {
		for _, s := range catalog.Skills {
			if s.Name == name {
				return testsupport.Equal(s.Repo.ID, repo)
			}
		}
		return fmt.Errorf("skill %s not found in catalog", name)
	})
	sc.Step(`^catalog owner of "([^"]*)" path "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, repo, p, want string) error {
		owner := catalog.Owner(ctx, repo, p)
		if owner == nil {
			return fmt.Errorf("expected owner %s for %s %s", want, repo, p)
		}
		return testsupport.Equal(owner.Name, want)
	})
	sc.Step(`^catalog owner of "([^"]*)" path "([^"]*)" is not found$`, func(ctx context.Context, repo, p string) error {
		if owner := catalog.Owner(ctx, repo, p); owner != nil {
			return fmt.Errorf("expected no owner for %s %s, got %s", repo, p, owner.Name)
		}
		return nil
	})
}
