package issues

// ConfigIssue is one problem in the configuration (after it was parsed),
// e.g. an invalid tag expression or a source listed twice.
type ConfigIssue struct {
	// Code is the kind of problem, e.g. "invalid-tags", "duplicate-source".
	Code string
	// Source is the source the setting belongs to (model.SourceKey.String()),
	// empty for a setting outside sources.
	Source string
	// Setting is where in the configuration the problem is, e.g.
	// "sources[1].tags".
	Setting string
	Message string
}

var _ Reportable = ConfigIssue{}

// Report places the problem as source -> setting, leaving out the source
// for a setting outside sources.
//
// Пример: {Source "local:repo", Setting "sources[1].tags"} -> Where =
// [source "local:repo", setting "sources[1].tags"]; {Setting "targets[0]"}
// -> [setting "targets[0]"].
func (e ConfigIssue) Report() IssueReportRow {
	return IssueReportRow{Code: e.Code, Message: e.Message, Where: where(
		Location{LocationSource, e.Source},
		Location{LocationSetting, e.Setting},
	)}
}

func (e ConfigIssue) Error() string { return errorText(e.Report()) }

type ConfigIssues []ConfigIssue

func (e ConfigIssues) Error() string { return joinErrors(e) }
