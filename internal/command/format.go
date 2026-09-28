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
//	└── skill guide (a/guide)
//	    ├── duplicate-name
//	    │   also defined at local:/p/other guide
//	    └── file SKILL.md
//	        └── link [x](./gone.md)
//	            └── missing-link-target
//	                link target does not exist
//	Found 2 problem(s)
func PrintIssues(out io.Writer, list []issues.Reportable, colored bool) {
	rows := make([]issues.IssueReportRow, 0, len(list))
	for _, i := range list {
		rows = append(rows, i.Report())
	}
	// Sort locations while preserving the validators' order for problems at
	// the same location.
	slices.SortStableFunc(rows, func(a, b issues.IssueReportRow) int {
		return slices.CompareFunc(a.Where, b.Where, func(x, y issues.Location) int {
			if c := strings.Compare(string(x.Kind), string(y.Kind)); c != 0 {
				return c
			}
			return strings.Compare(x.Value, y.Value)
		})
	})
	root := &issueNode{}
	for _, row := range rows {
		node := root
		for _, location := range row.Where {
			child := node.child(location)
			if child == nil {
				child = &issueNode{location: location}
				node.children = append(node.children, child)
			}
			node = child
		}
		node.rows = append(node.rows, row)
	}
	for i, row := range root.rows {
		printIssueRow(out, "", i == len(root.rows)-1, row, colored)
	}
	for _, node := range root.children {
		fmt.Fprintln(out, issueLocationText(node.location, colored))
		printIssueContents(out, node, "", colored)
	}
	fmt.Fprintf(out, "Found %d problem(s)\n", len(rows))
}

type issueNode struct {
	location issues.Location
	children []*issueNode
	rows     []issues.IssueReportRow
}

func (n *issueNode) child(location issues.Location) *issueNode {
	for _, child := range n.children {
		if child.location == location {
			return child
		}
	}
	return nil
}

func printIssueContents(out io.Writer, node *issueNode, prefix string, color bool) {
	total := len(node.rows) + len(node.children)
	entry := 0
	for _, row := range node.rows {
		printIssueRow(out, prefix, entry == total-1, row, color)
		entry++
	}
	for _, child := range node.children {
		last := entry == total-1
		branch := "├── "
		continuation := "│   "
		if last {
			branch = "└── "
			continuation = "    "
		}
		fmt.Fprintf(out, "%s%s%s\n", prefix, branch, issueLocationText(child.location, color))
		printIssueContents(out, child, prefix+continuation, color)
		entry++
	}
}

func printIssueRow(out io.Writer, prefix string, last bool, row issues.IssueReportRow, color bool) {
	branch := "├── "
	continuation := "│   "
	if last {
		branch = "└── "
		continuation = "    "
	}
	fmt.Fprintf(out, "%s%s%s\n", prefix, branch, ansi(row.Code, "1;31", color))
	fmt.Fprintf(out, "%s%s%s\n", prefix, continuation, row.Message)
}

func issueLocationText(location issues.Location, color bool) string {
	kind := ansi(string(location.Kind), "90", color)
	value := ansi(location.Value, "1", color)
	return kind + " " + value
}

func ansi(text, code string, enabled bool) string {
	if !enabled {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
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
