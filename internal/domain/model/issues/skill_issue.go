package issues

// SkillIssue is one problem found while loading or validating skills,
// with enough context to report it grouped by source, skill and file.
type SkillIssue struct {
	// Code is the kind of problem, e.g. "nested-skill", "missing-link-target".
	Code string
	// Source is the source the skill comes from (model.SourceKey.String()).
	Source string
	// Skill is the skill's name; SkillPath is its folder (or, for a flat
	// skill, its marker file) from the repository folder.
	Skill, SkillPath string
	// File is the file the problem is in; Link is the link's text as
	// written, when the problem is a link.
	File, Link string
	Message    string
}

var _ Reportable = SkillIssue{}

// Report places the problem as source -> skill -> file -> link, leaving out
// the levels it has no value for.
//
// Пример: {Source "local:repo", Skill "guide", SkillPath "a/guide", File
// "docs/x.md", Link "[x](./y.md)"} -> Where = [source "local:repo", skill
// "guide (a/guide)", file "docs/x.md", link "[x](./y.md)"]; без Skill и
// Link -> [source "local:repo", file "docs/x.md"].
func (e SkillIssue) Report() IssueReportRow {
	skill := e.Skill
	if skill != "" && e.SkillPath != "" {
		skill += " (" + e.SkillPath + ")"
	}
	return IssueReportRow{Code: e.Code, Message: e.Message, Where: where(
		Location{LocationSource, e.Source},
		Location{LocationSkill, skill},
		Location{LocationFile, e.File},
		Location{LocationLink, e.Link},
	)}
}

func (e SkillIssue) Error() string { return errorText(e.Report()) }

type SkillIssues []SkillIssue

func (e SkillIssues) Error() string { return joinErrors(e) }

// Problem is a SkillIssue with only a code and a message, as an error.
func Problem(code, message string) error { return SkillIssue{Code: code, Message: message} }
