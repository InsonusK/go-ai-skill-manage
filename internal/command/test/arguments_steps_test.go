package command_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/command"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

func argumentSteps(sc *godog.ScenarioContext) {
	var actual command.Invocation
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
	// parsed global options are: the Global options as JSON.
	sc.Step(`^parsed global options are$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		return testsupport.JSON(actual.Global, d)
	})
	// parsed command options are: the command's name and its options (its struct)
	// as JSON.
	sc.Step(`^parsed command "([^"]*)" options are$`, func(ctx context.Context, name string, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		if actual.Command == nil || actual.Command.Name() != name {
			return fmt.Errorf("command=%v want %s", actual.Command, name)
		}
		return testsupport.JSON(actual.Command, d)
	})
	sc.Step(`^argument error contains "(.*)"$`, func(ctx context.Context, want string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %s", failure, want)
		}
		return nil
	})
}
