package filesystem_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"os"
	"path/filepath"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var dir string
	var failure error
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if dir != "" {
			return ctx, os.RemoveAll(dir)
		}
		return ctx, nil
	})
	sc.Step(`^an empty target$`, func(ctx context.Context) error {
		var err error
		dir, err = os.MkdirTemp("", "aism-test-")
		testsupport.Log("target=%s", dir)
		return err
	})
	sc.Step(`^target fixture file "([^"]*)" contains "([^"]*)"$`, func(ctx context.Context, p, data string) error {
		return os.WriteFile(filepath.Join(dir, p), []byte(data), 0644)
	})
	sc.Step(`^target symlink "([^"]*)" points to "([^"]*)"$`, func(ctx context.Context, p, dest string) error { return os.Symlink(dest, filepath.Join(dir, p)) })
	sc.Step(`^snapshot marks "([^"]*)" as existing unmanaged entry$`, func(ctx context.Context, p string) error {
		state, err := (filesystem.Store{}).Snapshot(ctx, dir)
		if err != nil {
			return err
		}
		return testsupport.Equal([]bool{state[p].Exists, state[p].Managed}, []bool{true, false})
	})
	sc.Step(`^unmanaged target directory "([^"]*)"$`, func(ctx context.Context, p string) error {
		testsupport.Log("unmanaged=%s", p)
		return os.MkdirAll(filepath.Join(dir, p), 0755)
	})
	sc.Step(`^I apply a "([^"]*)" operation for "([^"]*)" containing "([^"]*)"$`, func(ctx context.Context, action, name, body string) error {
		op := model.Operation{Name: name, Action: action, Hash: "hash", Files: []model.OutputFile{{Path: "SKILL.md", Data: []byte(body), Mode: 0644}}}
		failure = (filesystem.Store{}).Apply(ctx, model.TargetPlan{Target: model.Target{Path: dir}, Operations: []model.Operation{op}})
		testsupport.Log("apply=%s %s error=%v", action, name, failure)
		return nil
	})
	sc.Step(`^target file "([^"]*)" equals "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		if failure != nil {
			return failure
		}
		raw, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			return err
		}
		return testsupport.Equal(string(raw), want)
	})
	sc.Step(`^state for "([^"]*)" is managed with hash "([^"]*)"$`, func(ctx context.Context, name, hash string) error {
		state, err := (filesystem.Store{}).Snapshot(ctx, dir)
		if err != nil {
			return err
		}
		s := state[name]
		return testsupport.Equal([]any{s.Managed, s.Hash, s.HasMain, s.Version}, []any{true, hash, true, model.TransformVersion})
	})
	sc.Step(`^target path "([^"]*)" exists "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		_, err := os.Stat(filepath.Join(dir, p))
		return testsupport.Equal(err == nil, want == "true")
	})
	sc.Step(`^filesystem error contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), s) {
			return fmt.Errorf("error=%v want %s", failure, s)
		}
		return nil
	})
}
