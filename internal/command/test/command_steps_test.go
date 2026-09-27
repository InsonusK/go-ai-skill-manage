package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/command"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// unescape turns \n into a line break and \" into a quote in a step's text.
func unescape(s string) string { return strings.NewReplacer(`\n`, "\n", `\"`, `"`).Replace(s) }

func initialize(sc *godog.ScenarioContext) {
	argumentSteps(sc)
	var dir string
	var code int
	var stdout, stderr bytes.Buffer
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		entity.SetDefaultLinkSearcher(links.NewDefaultLinkFactory())
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		entity.SetDefaultLinkSearcher(nil)
		return ctx, os.RemoveAll(dir)
	})
	sc.Step(`^CLI project$`, func(ctx context.Context, d *godog.DocString) error {
		var err error
		dir, err = os.MkdirTemp("", "aism-cli-test-")
		if err != nil {
			return err
		}
		var files map[string]string
		if err := json.Unmarshal([]byte(d.Content), &files); err != nil {
			return err
		}
		for p, v := range files {
			target := filepath.Join(dir, p)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(target, []byte(v), 0o644); err != nil {
				return err
			}
		}
		testsupport.Log("project=%s files=%v", dir, files)
		return nil
	})
	sc.Step(`^project file "([^"]*)" is removed$`, func(ctx context.Context, p string) error {
		return os.RemoveAll(filepath.Join(dir, p))
	})
	runCommand := func(ctx context.Context, args []string) error {
		stdout.Reset()
		stderr.Reset()
		opts, err := command.Parse(args)
		if err != nil {
			code = 2
			fmt.Fprintln(&stderr, err)
			return nil
		}
		previous := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&stderr, nil)))
		defer slog.SetDefault(previous)
		store := filesystem.Store{}
		app := command.App{Providers: map[string]interfaces.SourceProvider{"local": repository.Local{}}, State: store, Writer: store, ReadFile: os.ReadFile, Out: &stdout, Err: &stderr, Version: "test-version"}
		code = app.Execute(ctx, opts, dir)
		testsupport.Log("exit=%d\nstdout=%s\nstderr=%s", code, stdout.String(), stderr.String())
		return nil
	}
	sc.Step(`^I run arguments "([^"]*)"$`, func(ctx context.Context, s string) error { return runCommand(ctx, strings.Fields(s)) })
	sc.Step(`^I run argv$`, func(ctx context.Context, d *godog.DocString) error {
		var args []string
		if err := json.Unmarshal([]byte(d.Content), &args); err != nil {
			return err
		}
		return runCommand(ctx, args)
	})

	sc.Step(`^exit code is "([^"]*)"$`, func(ctx context.Context, s string) error { return testsupport.Equal(strconv.Itoa(code), s) })
	sc.Step(`^stdout contains "(.*)"$`, func(ctx context.Context, s string) error {
		if !strings.Contains(stdout.String(), unescape(s)) {
			return fmt.Errorf("stdout lacks %q:\n%s", unescape(s), stdout.String())
		}
		return nil
	})
	// console: stdout and stderr, warnings from the logs included; \n is a
	// line break.
	sc.Step(`^console contains "(.*)"$`, func(ctx context.Context, s string) error {
		all := stdout.String() + stderr.String()
		if !strings.Contains(all, unescape(s)) {
			return fmt.Errorf("console lacks %q:\n%s", unescape(s), all)
		}
		return nil
	})
	sc.Step(`^console shows "(.*)" once$`, func(ctx context.Context, s string) error {
		all := stdout.String() + stderr.String()
		return testsupport.Equal(strings.Count(all, unescape(s)), 1)
	})
	sc.Step(`^project file "([^"]*)" contains "(.*)"$`, func(ctx context.Context, p, want string) error {
		raw, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			return err
		}
		testsupport.Log("file=%s content=%s", p, raw)
		if !strings.Contains(string(raw), unescape(want)) {
			return fmt.Errorf("file %s lacks %q:\n%s", p, unescape(want), raw)
		}
		return nil
	})
	sc.Step(`^project path "([^"]*)" exists "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		_, err := os.Stat(filepath.Join(dir, p))
		return testsupport.Equal(err == nil, want == "true")
	})
}
