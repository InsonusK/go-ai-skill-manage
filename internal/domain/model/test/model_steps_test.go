package model_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
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
	sc.Step(`^a skill "([^"]*)" rooted at "([^"]*)" main "([^"]*)" with nested file "([^"]*)" containing "([^"]*)"$`, func(ctx context.Context, name, skillDirPath, mainFilePath, nestedFilePath, content string) error {
		lazyCounts = map[string]int{}
		files := fstest.MapFS{model.NestedRepoPath(skillDirPath, nestedFilePath): &fstest.MapFile{Data: []byte(content)}}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: countingFS{files: files, counts: lazyCounts}}
		lazySkill = &model.Skill{Name: name, MainFilePath: mainFilePath, SkillDirPath: skillDirPath, Format: model.AgentDirSkill, Repo: repo, Files: []model.File{{Path: nestedFilePath}}}
		lazyIndex = map[string]int{nestedFilePath: 0}
		lazyData, lazyErr = nil, nil
		return nil
	})
	sc.Step(`^a skill "([^"]*)" rooted at "([^"]*)" main "([^"]*)" with a missing nested file "([^"]*)"$`, func(ctx context.Context, name, skillDirPath, mainFilePath, nestedFilePath string) error {
		lazyCounts = map[string]int{}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: countingFS{files: fstest.MapFS{}, counts: lazyCounts}}
		lazySkill = &model.Skill{Name: name, MainFilePath: mainFilePath, SkillDirPath: skillDirPath, Format: model.AgentDirSkill, Repo: repo, Files: []model.File{{Path: nestedFilePath}}}
		lazyIndex = map[string]int{nestedFilePath: 0}
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
		return testsupport.Equal(strconv.Itoa(lazyCounts[model.NestedRepoPath(lazySkill.SkillDirPath, path)]), want)
	})
	sc.Step(`^reading nested file data fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if lazyErr == nil || !strings.Contains(lazyErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", lazyErr, want)
		}
		return nil
	})

	var pathFiles []*model.File
	var pathErr error
	var pathSnapshot int
	var dataResult []byte
	var dataErr error
	totalReads := func() int {
		n := 0
		for _, c := range lazyCounts {
			n += c
		}
		return n
	}
	sc.Step(`^a skill "([^"]*)" rooted at "([^"]*)" main "([^"]*)" with tree$`, func(ctx context.Context, name, root, main string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		lazyCounts = map[string]int{}
		files := fstest.MapFS{}
		for p, content := range raw {
			files[model.NestedRepoPath(root, p)] = &fstest.MapFile{Data: []byte(content)}
		}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: countingFS{files: files, counts: lazyCounts}}
		lazySkill = &model.Skill{Name: name, MainFilePath: main, SkillDirPath: root, Format: model.AgentDirSkill, Repo: repo}
		pathFiles, pathErr, dataResult, dataErr = nil, nil, nil, nil
		return nil
	})
	sc.Step(`^I list files by path "([^"]*)"(?: again)?$`, func(ctx context.Context, p string) error {
		pathFiles, pathErr = lazySkill.FilesByPath(p)
		return nil
	})
	sc.Step(`^I find files by path "([^"]*)" matching "([^"]*)"$`, func(ctx context.Context, p, pattern string) error {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		pathFiles, pathErr = lazySkill.Find(p, re)
		return nil
	})
	sc.Step(`^listed files are "([^"]*)"$`, func(ctx context.Context, want string) error {
		if pathErr != nil {
			return pathErr
		}
		var got []string
		for _, f := range pathFiles {
			got = append(got, f.Path)
		}
		return testsupport.Equal(strings.Join(got, ","), want)
	})
	sc.Step(`^listing files fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if pathErr == nil || !strings.Contains(pathErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", pathErr, want)
		}
		return nil
	})
	sc.Step(`^total directory reads so far are remembered$`, func(ctx context.Context) error {
		pathSnapshot = totalReads()
		return nil
	})
	sc.Step(`^no additional directories were read$`, func(ctx context.Context) error {
		return testsupport.Equal(strconv.Itoa(totalReads()), strconv.Itoa(pathSnapshot))
	})
	sc.Step(`^I read file data at "([^"]*)"$`, func(ctx context.Context, path string) error {
		files, err := lazySkill.FilesByPath("")
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.Path == path {
				dataResult, dataErr = lazySkill.Data(f)
				return nil
			}
		}
		return fmt.Errorf("file %q not found", path)
	})
	sc.Step(`^I read data of the first found file$`, func(ctx context.Context) error {
		dataResult, dataErr = pathFiles[0].Content()
		return nil
	})
	sc.Step(`^the first found file's data is "([^"]*)"$`, func(ctx context.Context, want string) error {
		if dataErr != nil {
			return dataErr
		}
		return testsupport.Equal(string(dataResult), want)
	})
	sc.Step(`^the first listed file's data is "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(string(pathFiles[0].Data), want)
	})
	sc.Step(`^file data at "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, path, want string) error {
		if dataErr != nil {
			return dataErr
		}
		return testsupport.Equal(string(dataResult), want)
	})

	var catalog *model.SkillCatalog
	var catalogErr error
	sc.Step(`^a catalog with conflict policy "([^"]*)"$`, func(ctx context.Context, policy string) error {
		catalog = &model.SkillCatalog{Conflict: policy}
		catalogErr = nil
		return nil
	})
	sc.Step(`^I add skill "([^"]*)" from repo "([^"]*)" main "([^"]*)"$`, func(ctx context.Context, name, repo, main string) error {
		skill := &model.Skill{Name: name, MainFilePath: main, SkillDirPath: main[:strings.LastIndex(main, "/")], Format: model.AgentDirSkill, Repo: &model.Repository{ID: repo}}
		catalogErr = catalog.GetOrAdd(ctx, skill)
		return nil
	})
	sc.Step(`^I add skill "([^"]*)" from repo "([^"]*)" main "([^"]*)" with files "([^"]*)"$`, func(ctx context.Context, name, repo, main, filesArg string) error {
		var files []model.File
		for _, p := range strings.Split(filesArg, ",") {
			files = append(files, model.File{Path: p})
		}
		skill := &model.Skill{Name: name, MainFilePath: main, SkillDirPath: main[:strings.LastIndex(main, "/")], Format: model.AgentDirSkill, Repo: &model.Repository{ID: repo}, Files: files}
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
