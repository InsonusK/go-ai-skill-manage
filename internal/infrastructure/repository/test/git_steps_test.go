package repository_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

type recorder struct{ args []string }

func (r *recorder) Run(ctx context.Context, args ...string) error { r.args = args; return nil }
func gitSteps(sc *godog.ScenarioContext) {
	var recorded recorder
	var failure error
	sc.Step(`^I prepare a clone for branch "([^"]*)"$`, func(ctx context.Context, branch string) error {
		testsupport.Log("branch=%s", branch)
		return (repository.GitCloner{Runner: &recorded}).Clone(ctx, "https://github.com/owner/repo.git", branch, "/tmp/dest")
	})
	sc.Step(`^git arguments are$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(recorded.args, d) })
	sc.Step(`^I run Git with "([^"]*)"$`, func(ctx context.Context, arg string) error {
		failure = (repository.GitProcess{}).Run(ctx, arg)
		testsupport.Log("git=%s error=%v", arg, failure)
		return nil
	})
	sc.Step(`^I run Git in a cancelled context$`, func(ctx context.Context) error {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		failure = (repository.GitProcess{}).Run(cancelled, "version")
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
