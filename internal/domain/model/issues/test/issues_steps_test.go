package issues_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	var list issues.SkillIssues

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
		list = nil
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
	sc.Step(`^the label of code "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, code, want string) error {
		return testsupport.Equal(issues.Code(code).Label(), want)
	})
	sc.Step(`^the severity of code "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, code, want string) error {
		return testsupport.Equal(string(issues.Code(code).Severity()), want)
	})
	sc.Step(`^the list has errors "(true|false)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(fmt.Sprint(issues.HasErrors(list)), want)
	})
	sc.Step(`^the warnings of the list are "([^"]*)"$`, func(ctx context.Context, want string) error {
		codes := []string{}
		for _, w := range issues.Warnings(list) {
			codes = append(codes, string(w.Code))
		}
		return testsupport.Equal(strings.Join(codes, ","), want)
	})
	sc.Step(`^every declared code has a unique number like "([^"]*)"$`, func(ctx context.Context, pattern string) error {
		form := regexp.MustCompile(pattern)
		seen := map[string]issues.Code{}
		for _, code := range issues.Codes() {
			id := code.ID()
			testsupport.Log("code=%s id=%s", code, id)
			if !form.MatchString(id) {
				return fmt.Errorf("code %s has number %q; want %s", code, id, pattern)
			}
			// The number is unique without its letter: E302 and W302 would
			// be searched as one.
			id = id[1:]
			if other, ok := seen[id]; ok {
				return fmt.Errorf("number %s is used by %s and %s", id, other, code)
			}
			seen[id] = code
		}
		if len(seen) == 0 {
			return fmt.Errorf("no codes are declared")
		}
		return nil
	})
	sc.Step(`^no Go file under "([^"]*)" writes an issue code as text$`, func(ctx context.Context, root string) error {
		literal := regexp.MustCompile(`(\bCode:\s*|\.Code(, [\w.]+)*\s*[=!]?=\s*|issues\.Problem\()"`)
		var found []string
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			content, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			for n, line := range strings.Split(string(content), "\n") {
				if literal.MatchString(line) {
					found = append(found, fmt.Sprintf("%s:%d", p, n+1))
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		testsupport.Log("root=%s found=%v", root, found)
		if len(found) > 0 {
			return fmt.Errorf("issue codes written as text, declare them in codes.go: %v", found)
		}
		return nil
	})
}
