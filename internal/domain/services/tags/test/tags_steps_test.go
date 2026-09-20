package tags_test

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/tags"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var input []string
	var matched bool
	var message string
	sc.Step(`^tags "([^"]*)"$`, func(ctx context.Context, s string) error {
		input = strings.Split(s, ",")
		testsupport.Log("tags=%v", input)
		return nil
	})
	sc.Step(`^I evaluate "([^"]*)"$`, func(ctx context.Context, s string) error {
		var err error
		matched, err = tags.Match(input, []string{s})
		message = ""
		if err != nil {
			message = err.Error()
		}
		testsupport.Log("evaluate=%q result=%v error=%s", s, matched, message)
		return nil
	})
	sc.Step(`^the match is "([^"]*)" and error is "([^"]*)"$`, func(ctx context.Context, want, err string) error {
		return testsupport.Equal([]any{matched, message}, []any{want == "true", err})
	})
}
