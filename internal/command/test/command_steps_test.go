package command_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/command"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	argumentSteps(sc)
	var dir string
	var code int
	var stdout, stderr bytes.Buffer
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
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
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(target, []byte(v), 0644); err != nil {
				return err
			}
		}
		testsupport.Log("project=%s files=%v", dir, files)
		return nil
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
		detector := discovery.Detector{Codec: document.Codec{}}
		store := filesystem.Store{}
		sources := repository.NewSourceManager(map[string]interfaces.SourceProvider{"local": repository.Local{}})
		service := &services.SyncService{Sources: sources, Detector: detector, Relations: relations.Expander{Detector: detector}, Planner: planning.Planner{State: store, Codec: document.Codec{}}, Writer: store}
		app := command.App{Sync: service, ReadFile: os.ReadFile, Out: &stdout, Err: &stderr, Version: "test-version"}
		code = app.Execute(ctx, opts, dir)
		testsupport.Log("exit=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
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
	sc.Step(`^stdout contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("stdout=%s", stdout.String())
		if !strings.Contains(stdout.String(), s) {
			return fmt.Errorf("missing %q", s)
		}
		return nil
	})
	sc.Step(`^console contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		all := stdout.String() + stderr.String()
		testsupport.Log("console=%s", all)
		if !strings.Contains(all, s) {
			return fmt.Errorf("missing %q", s)
		}
		return nil
	})
	sc.Step(`^project file "([^"]*)" contains "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		raw, err := os.ReadFile(filepath.Join(dir, p))
		if err != nil {
			return err
		}
		testsupport.Log("file=%s content=%s", p, raw)
		if !strings.Contains(string(raw), want) {
			return fmt.Errorf("file %s lacks %q", p, want)
		}
		return nil
	})
	sc.Step(`^project path "([^"]*)" exists "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		_, err := os.Stat(filepath.Join(dir, p))
		return testsupport.Equal(err == nil, want == "true")
	})
}
