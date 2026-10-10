package common

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

// PrintIssues prints problems of any kind as one tree: each problem under
// where it is (issues.Reportable), from the broadest place to the
// narrowest, problems at the same place together, then a count of errors
// and warnings. A problem is its number and code, its message and, one a
// line, its details.
//
// Пример:
//
//	source local:/p/skills
//	└── skill guide (a/guide)
//	    ├── E208 duplicate-name
//	    │   also defined at local:/p/other guide
//	    └── file SKILL.md
//	        └── link [x](./gone.md)
//	            └── E302 missing-link-target
//	                link target does not exist
//	└── file shared/notes.md
//	    └── W311 shared-file-links
//	        shared file outside skills is copied as written, check where its links lead:
//	        - [one](../one/SKILL.md)
//	        - [[two/SKILL.md|two]]
//	Found 2 error(s), 1 warning(s)
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
	warnings := 0
	for _, row := range rows {
		if row.Code.Severity() == issues.SeverityWarning {
			warnings++
		}
	}
	fmt.Fprintf(out, "Found %d error(s), %d warning(s)\n", len(rows)-warnings, warnings)
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
	style := "1;31"
	if row.Code.Severity() == issues.SeverityWarning {
		style = "1;33"
	}
	fmt.Fprintf(out, "%s%s%s\n", prefix, branch, ansi(row.Code.Label(), style, color))
	fmt.Fprintf(out, "%s%s%s\n", prefix, continuation, row.Message)
	for _, detail := range row.Details {
		fmt.Fprintf(out, "%s%s- %s\n", prefix, continuation, detail)
	}
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

// Rows is a list of problems of one kind as the printer takes them.
func Rows[T issues.Reportable](list []T) []issues.Reportable {
	out := make([]issues.Reportable, 0, len(list))
	for _, i := range list {
		out = append(out, i)
	}
	return out
}

// Reportables returns the problems err carries, nil if it carries none.
func Reportables(err error) []issues.Reportable {
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
