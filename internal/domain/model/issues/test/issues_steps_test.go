package issues_test

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func initialize(sc *godog.ScenarioContext) {
	var issue interface {
		issues.Reportable
		error
	}
	var errText string
	var row issues.IssueReportRow

	sc.Step(`^a skill issue (.+)$`, func(ctx context.Context, raw string) error {
		var i issues.SkillIssue
		testsupport.Log("issue=%s", raw)
		if err := json.Unmarshal([]byte(raw), &i); err != nil {
			return err
		}
		issue, errText = i, i.Error()
		return nil
	})
	sc.Step(`^a config issue (.+)$`, func(ctx context.Context, raw string) error {
		var i issues.ConfigIssue
		testsupport.Log("issue=%s", raw)
		if err := json.Unmarshal([]byte(raw), &i); err != nil {
			return err
		}
		issue, errText = i, i.Error()
		return nil
	})
	sc.Step(`^a target issue (.+)$`, func(ctx context.Context, raw string) error {
		var i issues.TargetIssue
		testsupport.Log("issue=%s", raw)
		if err := json.Unmarshal([]byte(raw), &i); err != nil {
			return err
		}
		issue, errText = i, i.Error()
		return nil
	})
	sc.Step(`^skill issues (.+)$`, func(ctx context.Context, raw string) error {
		var list issues.SkillIssues
		testsupport.Log("issues=%s", raw)
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			return err
		}
		errText = list.Error()
		return nil
	})
	sc.Step(`^I report the issue$`, func(ctx context.Context) error {
		row = issue.Report()
		return nil
	})
	sc.Step(`^the report row is (.+)$`, func(ctx context.Context, want string) error {
		return testsupport.JSON(row, &godog.DocString{Content: want})
	})
	sc.Step(`^the error text is "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(errText, strings.ReplaceAll(want, `\n`, "\n"))
	})
}
