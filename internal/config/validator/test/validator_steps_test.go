package validator_test

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/internal/config/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func initialize(sc *godog.ScenarioContext) {
	var problems issues.ConfigIssues

	// The configuration goes the real way: Parse, Resolve from "/project",
	// then Validate.
	sc.Step(`^I validate configuration$`, func(ctx context.Context, d *godog.DocString) error {
		testsupport.Log("config=%s", d.Content)
		c, err := config.Parse([]byte(d.Content))
		if err != nil {
			return err
		}
		req, err := config.Resolve(c, config.Overrides{}, "/project")
		if err != nil {
			return err
		}
		problems = validator.Validate(ctx, req)
		testsupport.Log("issues=%v", problems)
		return nil
	})
	sc.Step(`^there are no config issues$`, func(ctx context.Context) error {
		if len(problems) > 0 {
			return fmt.Errorf("unexpected issues:\n%v", problems)
		}
		return nil
	})
	// the config issues are: a JSON list of [code, source, setting].
	sc.Step(`^the config issues are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			got = append(got, []string{i.Code, i.Source, i.Setting})
		}
		return testsupport.JSON(got, d)
	})
	// The text runs to the last quote; \" inside it is a quote.
	// Only the issues with the given comma-separated codes, as above: a
	// feature checks its own validator's issues.
	sc.Step(`^the config issues with codes "([^"]*)" are$`, func(ctx context.Context, codes string, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			if slices.Contains(strings.Split(codes, ","), i.Code) {
				got = append(got, []string{i.Code, i.Source, i.Setting})
			}
		}
		return testsupport.JSON(got, d)
	})
	sc.Step(`^config issue (\d+) message contains "(.*)"$`, func(ctx context.Context, n int, text string) error {
		text = strings.ReplaceAll(text, `\"`, `"`)
		if n >= len(problems) || !strings.Contains(problems[n].Message, text) {
			return fmt.Errorf("issue %d of %v; want message with %q", n, problems, text)
		}
		return nil
	})
}
