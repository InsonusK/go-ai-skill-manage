package interfaces

import (
	"context"
	"errors"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// ErrSkillNotManaged: the target has no folder of that skill written by
// this tool (no folder, or no marker in it).
var ErrSkillNotManaged = errors.New("skill is not managed in the target")

// MarkerReader reads the marker of a skill written into a target --
// implemented by infrastructure/filesystem.Store.
type MarkerReader interface {
	// ReadMarker returns the marker of skill's folder in target, or an
	// error wrapping ErrSkillNotManaged.
	ReadMarker(ctx context.Context, target, skill string) (model.ManagedState, error)
}

// FeedbackDrafts keeps feedback drafts -- implemented by
// infrastructure/filesystem.FeedbackDrafts.
type FeedbackDrafts interface {
	// Create stores a new draft under id or, when taken, the first free
	// "id-N", and returns the id used.
	Create(ctx context.Context, id string, draft model.FeedbackDraft) (string, error)
	Load(ctx context.Context, id string) (model.FeedbackDraft, error)
	// List returns the ids of all stored drafts, of any status, sorted.
	List(ctx context.Context) ([]string, error)
	// Save overwrites an existing draft.
	Save(ctx context.Context, draft model.FeedbackDraft) error
}

// IssueTracker opens issues in a source's tracker -- implemented by
// infrastructure/tracker.GitHub.
type IssueTracker interface {
	// Create opens issue in source's tracker and returns its URL.
	Create(ctx context.Context, source model.SourceKey, issue model.NewIssue) (string, error)
}
