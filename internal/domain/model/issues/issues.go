// Package issues holds the problems found while checking a configuration
// or loading and validating skills. Each kind has its own fields, but all
// of them report themselves the same way (Reportable), so one printer can
// output any of them.
package issues

import (
	"fmt"
	"strings"
)

// Reportable is a problem that describes itself in the one format a
// printer understands, whatever its own fields are.
type Reportable interface {
	Report() IssueReportRow
}

// IssueReportRow is one problem in the printer's format: what went wrong
// (Code, Message) and where (Where), from the broadest place to the
// narrowest.
//
// Пример: ссылка в файле скила ->
// Where = [source "local:repo", skill "guide (a/guide)", file "docs/x.md",
// link "[x](./y.md)"]; ошибка конфига -> [source "local:repo", setting
// "sources[1].tags"].
type IssueReportRow struct {
	Code, Message string
	Where         []Location
}

// LocationKind names what a Location points at.
type LocationKind string

const (
	LocationSource  LocationKind = "source"
	LocationSkill   LocationKind = "skill"
	LocationFile    LocationKind = "file"
	LocationLink    LocationKind = "link"
	LocationSetting LocationKind = "setting"
)

// Location is one level of where a problem is.
type Location struct {
	Kind  LocationKind
	Value string
}

// where lists the non-empty locations in the given order.
func where(locations ...Location) []Location {
	out := []Location{}
	for _, l := range locations {
		if l.Value != "" {
			out = append(out, l)
		}
	}
	return out
}

// errorText is the one-line form of a row: "code: where...: message".
func errorText(row IssueReportRow) string {
	var at []string
	for _, l := range row.Where {
		at = append(at, l.Value)
	}
	if len(at) == 0 {
		return fmt.Sprintf("%s: %s", row.Code, row.Message)
	}
	return fmt.Sprintf("%s: %s: %s", row.Code, strings.Join(at, " "), row.Message)
}

// joinErrors is the Error text of a list: one line per problem.
func joinErrors[T Reportable](list []T) string {
	var out []string
	for _, i := range list {
		out = append(out, errorText(i.Report()))
	}
	return strings.Join(out, "\n")
}
