package model

import "time"

// FeedbackKind is what a feedback is about.
type FeedbackKind string

const (
	FeedbackBug         FeedbackKind = "bug"
	FeedbackImprovement FeedbackKind = "improvement"
)

// FeedbackStatus is where a feedback draft is: a draft becomes sent or
// declined, and neither changes again.
type FeedbackStatus string

const (
	FeedbackDraftStatus FeedbackStatus = "draft"
	FeedbackSent        FeedbackStatus = "sent"
	FeedbackDeclined    FeedbackStatus = "declined"
)

// FeedbackDraft is a feedback about a skill for its source's issue
// tracker. It is kept as a file in the project (committed, so it leaves a
// trace) and the file is the only source of truth: what the user edits
// there is what gets sent.
type FeedbackDraft struct {
	// ID names the draft (its file name without the extension).
	ID     string
	Status FeedbackStatus
	Kind   FeedbackKind
	// Skill is the skill's name, as written into a target.
	Skill string
	// Source, Commit and SkillPath come from the skill's marker
	// (ManagedState): where the skill was taken from.
	Source    SourceKey
	Commit    string
	SkillPath string
	CreatedAt time.Time
	Title     string
	Body      string
	// IssueURL and SentAt are set once the draft is sent.
	IssueURL string
	SentAt   time.Time
}

// NewIssue is an issue to open in a source's tracker.
type NewIssue struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels"`
}
