package issues

// TargetIssue is one problem with a target folder that stops writing into
// it, e.g. a folder of the same name as a skill that this tool didn't
// write.
type TargetIssue struct {
	// Code is the kind of problem, e.g. "unmanaged-target".
	Code string
	// Target is the target folder (model.Target.Path).
	Target string
	// Skill is the skill whose folder in the target has the problem.
	Skill   string
	Message string
}

var _ Reportable = TargetIssue{}

// Report places the problem as target -> skill.
//
// Пример: {Target "/p/.claude/skills", Skill "guide"} -> Where =
// [target "/p/.claude/skills", skill "guide"].
func (e TargetIssue) Report() IssueReportRow {
	return IssueReportRow{Code: e.Code, Message: e.Message, Where: where(
		Location{LocationTarget, e.Target},
		Location{LocationSkill, e.Skill},
	)}
}

func (e TargetIssue) Error() string { return errorText(e.Report()) }

type TargetIssues []TargetIssue

func (e TargetIssues) Error() string { return joinErrors(e) }
