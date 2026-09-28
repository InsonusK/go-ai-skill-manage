package repository_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

// recorder records git calls; rev-parse answers with a fixed commit.
type recorder struct{ calls [][]string }

func (r *recorder) Run(ctx context.Context, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	if len(args) > 0 && args[len(args)-2] == "rev-parse" {
		return "c0ffee\n", nil
	}
	return "", nil
}
func gitSteps(sc *godog.ScenarioContext) {
	var recorded recorder
	var commit, output string
	var failure error
	sc.Step(`^I prepare a clone for branch "([^"]*)"$`, func(ctx context.Context, branch string) error {
		testsupport.Log("branch=%s", branch)
		recorded = recorder{}
		var err error
		commit, err = (repository.GitCloner{Runner: &recorded}).Clone(ctx, "https://github.com/owner/repo.git", branch, "/tmp/dest")
		return err
	})
	sc.Step(`^git calls are$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(recorded.calls, d) })
	sc.Step(`^the cloned commit is "([^"]*)"$`, func(ctx context.Context, want string) error { return testsupport.Equal(commit, want) })
	sc.Step(`^I run Git with "([^"]*)"$`, func(ctx context.Context, arg string) error {
		output, failure = (repository.GitProcess{}).Run(ctx, arg)
		testsupport.Log("git=%s output=%q error=%v", arg, output, failure)
		return nil
	})
	sc.Step(`^git output starts with "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.HasPrefix(output, want), true)
	})
	sc.Step(`^I run Git in a cancelled context$`, func(ctx context.Context) error {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, failure = (repository.GitProcess{}).Run(cancelled, "version")
		testsupport.Log("error=%v", failure)
		return nil
	})
	sc.Step(`^git process error contains "([^"]*)"$`, func(ctx context.Context, w string) error {
		if w == "" {
			return failure
		}
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), w) {
			return fmt.Errorf("error=%v want %s", failure, w)
		}
		return nil
	})
}
