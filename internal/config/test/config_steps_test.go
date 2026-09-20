package config_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var input string
	var actual map[string]any
	var failure error
	sc.Step(`^configuration$`, func(ctx context.Context, d *godog.DocString) error {
		input = d.Content
		testsupport.Log("config=%s", input)
		return nil
	})
	sc.Step(`^I resolve configuration$`, func(ctx context.Context) error {
		c, err := config.Parse([]byte(input))
		failure = err
		if err != nil {
			return nil
		}
		req, err := config.Resolve(c, config.Overrides{}, "/project")
		failure = err
		if err != nil {
			return nil
		}
		sources := []any{}
		for _, s := range req.Sources {
			sources = append(sources, map[string]any{"type": s.Type, "path": s.Path, "subpaths": s.Subpaths, "tags": s.Tags, "skip": s.SkipFolders})
		}
		targets := []any{}
		for _, t := range req.Targets {
			targets = append(targets, map[string]any{"name": t.Name, "path": t.Path, "adapters": t.Adapters})
		}
		actual = map[string]any{"sources": sources, "targets": targets, "tempDir": req.TempDir, "dry": req.DryRun, "orphans": req.RemoveOrphans, "relations": req.AddRelations, "conflict": req.Conflict, "linkSkip": req.LinkSkipFolders}
		return nil
	})
	sc.Step(`^the configuration is$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		return testsupport.JSON(actual, d)
	})
	sc.Step(`^the config error contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), s) {
			return fmt.Errorf("error=%v; want %s", failure, s)
		}
		return nil
	})
}
