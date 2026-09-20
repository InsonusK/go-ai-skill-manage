package model_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var ownRoot, ownMain string
	var ownFlat bool
	sc.Step(`^a skill rooted at "([^"]*)" main "([^"]*)" flat "([^"]*)"$`, func(ctx context.Context, root, main, flat string) error {
		ownRoot, ownMain, ownFlat = root, main, flat == "true"
		return nil
	})
	sc.Step(`^path "([^"]*)" is owned "([^"]*)" and relative is "([^"]*)"$`, func(ctx context.Context, p, owned, relative string) error {
		gotOwned := model.OwnsPath(ownMain, ownRoot, ownFlat, p)
		gotRelative := model.RelativePath(ownMain, ownRoot, p)
		return testsupport.Equal([]any{gotOwned, gotRelative}, []any{owned == "true", relative})
	})

	type entryJSON struct {
		Name, Source, Root, Main string
		Flat                     bool
	}
	var entries []model.SkillEntry
	var originals map[string]map[string]string
	var skillMap *model.SkillMap
	var buildErr error
	sc.Step(`^skill entries$`, func(ctx context.Context, d *godog.DocString) error {
		var raw []entryJSON
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		entries = nil
		for _, e := range raw {
			entries = append(entries, model.SkillEntry{Name: e.Name, SourceKey: e.Source, Root: e.Root, Main: e.Main, Flat: e.Flat, Dest: e.Name})
		}
		return nil
	})
	sc.Step(`^skill file destinations$`, func(ctx context.Context, d *godog.DocString) error {
		return json.Unmarshal([]byte(d.Content), &originals)
	})
	sc.Step(`^I build the skill map$`, func(ctx context.Context) error {
		skillMap, buildErr = model.NewSkillMap(entries, originals)
		return nil
	})
	sc.Step(`^owner of "([^"]*)" path "([^"]*)" is skill "([^"]*)" at "([^"]*)"$`, func(ctx context.Context, source, p, name, dest string) error {
		entry, got, ok := skillMap.Owner(source, p)
		if !ok {
			return fmt.Errorf("owner not found for %s %s", source, p)
		}
		return testsupport.Equal([]any{entry.Name, got}, []any{name, dest})
	})
	sc.Step(`^owner of "([^"]*)" path "([^"]*)" is not found$`, func(ctx context.Context, source, p string) error {
		if _, _, ok := skillMap.Owner(source, p); ok {
			return fmt.Errorf("expected no owner for %s %s", source, p)
		}
		return nil
	})
	sc.Step(`^building the skill map fails with "([^"]*)"$`, func(ctx context.Context, contains string) error {
		if buildErr == nil || !strings.Contains(buildErr.Error(), contains) {
			return fmt.Errorf("error=%v want contains %s", buildErr, contains)
		}
		return nil
	})

	var catalog *model.Catalog
	var catalogErr error
	sc.Step(`^a catalog with conflict policy "([^"]*)"$`, func(ctx context.Context, policy string) error {
		catalog = &model.Catalog{Conflict: policy}
		catalogErr = nil
		return nil
	})
	sc.Step(`^I add skill "([^"]*)" from repo "([^"]*)" main "([^"]*)"$`, func(ctx context.Context, name, repo, main string) error {
		skill := &model.Skill{Name: name, Main: main, Root: main[:strings.LastIndex(main, "/")], Repo: &model.Repository{ID: repo}}
		catalogErr = catalog.Add(ctx, skill)
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
