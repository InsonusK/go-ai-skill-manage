package profiling_test

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/profiling"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var dir, file string
	var failure error
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		var err error
		dir, err = os.MkdirTemp("", "aism-prof-test-")
		file = filepath.Join(dir, "cpu.prof")
		return ctx, err
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		return ctx, os.RemoveAll(dir)
	})
	sc.Step(`^I record a CPU profile$`, func(ctx context.Context) error {
		stop, err := profiling.Start(file)
		if err != nil {
			return err
		}
		testsupport.Log("profile=%s", file)
		return stop()
	})
	sc.Step(`^the profile is a readable gzip stream$`, func(ctx context.Context) error {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gz.Close()
		data, err := io.ReadAll(gz)
		if err != nil {
			return err
		}
		testsupport.Log("profile bytes=%d", len(data))
		if len(data) == 0 {
			return fmt.Errorf("empty profile payload")
		}
		return nil
	})
	sc.Step(`^I start profiling in a missing directory$`, func(ctx context.Context) error {
		_, failure = profiling.Start(filepath.Join(dir, "missing/cpu.prof"))
		testsupport.Log("error=%v", failure)
		return nil
	})
	sc.Step(`^profiling error contains "([^"]*)"$`, func(ctx context.Context, want string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %s", failure, want)
		}
		return nil
	})
}
