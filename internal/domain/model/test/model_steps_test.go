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
}
