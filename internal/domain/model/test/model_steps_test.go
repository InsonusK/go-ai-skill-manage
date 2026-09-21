package model_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

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
