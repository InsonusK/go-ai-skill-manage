package model_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func registerPathSteps(sc *godog.ScenarioContext) {
	var read model.PathInRepo
	var readErr error
	sc.Step(`^the written path "([^"]*)" is detected as "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		return testsupport.Equal(string(model.DetectPathKind(p)), want)
	})
	sc.Step(`^I read the path "([^"]*)" written as "([^"]*)" from "([^"]*)" in repository "([^"]*)"$`, func(ctx context.Context, p, kind, base, repoDir string) error {
		read, readErr = model.MakePathInRepo(repoDir, p, model.PathKind(kind), base)
		testsupport.Log("path=%+v error=%v", read, readErr)
		return nil
	})
	sc.Step(`^the path written as "([^"]*)" from "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, kind, base, want string) error {
		if readErr != nil {
			return readErr
		}
		got, err := read.Path(model.PathKind(kind), base)
		if err != nil {
			return err
		}
		return testsupport.Equal(got, want)
	})
	sc.Step(`^writing the path as "([^"]*)" fails with "([^"]*)"$`, func(ctx context.Context, kind, want string) error {
		if readErr != nil {
			return readErr
		}
		_, err := read.Path(model.PathKind(kind), "")
		if err == nil || !strings.Contains(err.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", err, want)
		}
		return nil
	})
	sc.Step(`^reading the path fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if readErr == nil || !strings.Contains(readErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", readErr, want)
		}
		return nil
	})
}
