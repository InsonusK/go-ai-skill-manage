package config_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"log/slog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var input string
	var actual map[string]any
	var failure error
	var logs bytes.Buffer
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		logs.Reset()
		return ctx, nil
	})
	sc.Step(`^configuration$`, func(ctx context.Context, d *godog.DocString) error {
		input = d.Content
		testsupport.Log("config=%s", input)
		return nil
	})
	sc.Step(`^I resolve configuration$`, func(ctx context.Context) error {
		previous := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
		c, err := config.Parse([]byte(input))
		slog.SetDefault(previous)
		testsupport.Log("logs=%s", logs.String())
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
			sources = append(sources, map[string]any{"type": s.Type, "path": s.Path, "subpaths": s.Subpaths, "tags": s.Tags, "exclude": s.ExcludeFromChecks})
		}
		targets := []any{}
		for _, t := range req.Targets {
			targets = append(targets, map[string]any{"name": t.Name, "path": t.Path, "adapters": t.Adapters})
		}
		actual = map[string]any{"sources": sources, "targets": targets, "tempDir": req.TempDir, "dry": req.DryRun, "orphans": req.RemoveOrphans, "relations": req.AddRelations, "conflict": req.Conflict, "exclude": req.ExcludeFromChecks}
		return nil
	})
	sc.Step(`^the configuration is$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		return testsupport.JSON(actual, d)
	})
	sc.Step(`^the config log has (INFO|WARN) "([^"]*)"$`, func(ctx context.Context, level, text string) error {
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "level="+level) && strings.Contains(line, text) {
				return nil
			}
		}
		return fmt.Errorf("no %s log with %q in:\n%s", level, text, logs.String())
	})
	sc.Step(`^the config log has no (INFO|WARN)$`, func(ctx context.Context, level string) error {
		if strings.Contains(logs.String(), "level="+level) {
			return fmt.Errorf("unexpected %s log in:\n%s", level, logs.String())
		}
		return nil
	})
	sc.Step(`^the config error contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), s) {
			return fmt.Errorf("error=%v; want %s", failure, s)
		}
		return nil
	})
}
