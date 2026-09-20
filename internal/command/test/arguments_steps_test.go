package command_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/command"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func argumentSteps(sc *godog.ScenarioContext) {
	var actual command.Options
	var failure error
	sc.Step(`^I parse argument list$`, func(ctx context.Context, d *godog.DocString) error {
		var args []string
		if err := json.Unmarshal([]byte(d.Content), &args); err != nil {
			return err
		}
		actual, failure = command.Parse(args)
		testsupport.Log("args=%v error=%v", args, failure)
		return nil
	})
	sc.Step(`^parsed options are$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		o := actual.Override
		return testsupport.JSON(map[string]any{"config": actual.Config, "type": actual.SourceType, "path": actual.SourcePath, "subpaths": actual.Subpaths, "target": o.Target, "dry": o.DryRun, "force": o.Force, "orphans": o.RemoveOrphans, "relations": o.AddRelations, "debug": actual.Debug, "profile": actual.Profile, "profileOutput": actual.ProfileOutput}, d)
	})
	sc.Step(`^argument error contains "([^"]*)"$`, func(ctx context.Context, want string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %s", failure, want)
		}
		return nil
	})
}
