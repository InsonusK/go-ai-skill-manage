package command

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
)

// PrintResult prints what a sync did (or, for a dry run, would do): each
// target with its operations, then a summary.
//
// Пример:
//
//	Target claude: /p/.claude/skills
//	  create review
//	  update guide
//	Synced 2 skill(s) to 1 target(s)
func PrintResult(out io.Writer, result handler.SyncResult) {
	for _, plan := range result.Plans {
		fmt.Fprintf(out, "Target %s: %s\n", plan.Target.Name, plan.Target.Path)
		for _, op := range plan.Operations {
			fmt.Fprintf(out, "  %s %s\n", op.Action, op.Name)
		}
	}
	if result.DryRun {
		fmt.Fprintf(out, "Dry run: %d skill(s), %d target(s); nothing written\n", len(result.Skills), len(result.Plans))
		return
	}
	fmt.Fprintf(out, "Synced %d skill(s) to %d target(s)\n", len(result.Skills), len(result.Plans))
}

// PrintIssues prints problems of any kind as one tree: each problem under
// where it is (issues.Reportable), from the broadest place to the
// narrowest, problems at the same place together, then a count.
//
// Пример:
//
//	source local:/p/skills
//	  skill guide (a/guide)
//	    duplicate-name: also defined at local:/p/other guide
//	    file SKILL.md
//	      link [x](./gone.md)
//	        missing-link-target: link target does not exist
//	Found 2 problem(s)
func PrintIssues(out io.Writer, list []issues.Reportable) {
	rows := make([]issues.IssueReportRow, 0, len(list))
	for _, i := range list {
		rows = append(rows, i.Report())
	}
	// Group by place; rows at the same place keep their order.
	slices.SortStableFunc(rows, func(a, b issues.IssueReportRow) int {
		return slices.CompareFunc(a.Where, b.Where, func(x, y issues.Location) int {
			if c := strings.Compare(string(x.Kind), string(y.Kind)); c != 0 {
				return c
			}
			return strings.Compare(x.Value, y.Value)
		})
	})
	var shown []issues.Location
	for _, row := range rows {
		common := 0
		for common < len(shown) && common < len(row.Where) && shown[common] == row.Where[common] {
			common++
		}
		for depth := common; depth < len(row.Where); depth++ {
			fmt.Fprintf(out, "%s%s %s\n", strings.Repeat("  ", depth), row.Where[depth].Kind, row.Where[depth].Value)
		}
		shown = row.Where
		fmt.Fprintf(out, "%s%s: %s\n", strings.Repeat("  ", len(row.Where)), row.Code, row.Message)
	}
	fmt.Fprintf(out, "Found %d problem(s)\n", len(rows))
}

// reportables returns the problems err carries, nil if it carries none.
func reportables(err error) []issues.Reportable {
	var out []issues.Reportable
	var skills issues.SkillIssues
	var configs issues.ConfigIssues
	var targets issues.TargetIssues
	switch {
	case errors.As(err, &skills):
		for _, i := range skills {
			out = append(out, i)
		}
	case errors.As(err, &configs):
		for _, i := range configs {
			out = append(out, i)
		}
	case errors.As(err, &targets):
		for _, i := range targets {
			out = append(out, i)
		}
	}
	return out
}
