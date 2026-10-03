package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/feedback"
)

// FeedbackService takes a feedback about a skill to the issue tracker of
// the skill's source in three stages, so that only the user sends it:
// Draft stores it as a file in the project, Preview shows exactly what
// would be sent (with its hash), Send opens the issue only for that hash.
// Pending previews all drafts not yet sent or declined.
// Decline closes a draft without sending.
type FeedbackService struct {
	Markers interfaces.MarkerReader
	Drafts  interfaces.FeedbackDrafts
	// Trackers are the issue trackers by source type (model.SourceKey.Type).
	Trackers map[string]interfaces.IssueTracker
	Now      func() time.Time
	// Version is the CLI version, written into every issue.
	Version string
}

// FeedbackInput is what the agent writes about a skill.
type FeedbackInput struct {
	Skill       string
	Kind        model.FeedbackKind
	Title, Body string
}

// FeedbackPreview is a draft and exactly what sending it would open:
// Hash is to be passed to Send.
type FeedbackPreview struct {
	Draft model.FeedbackDraft
	Issue model.NewIssue
	Hash  string
}

// Draft stores in's feedback as a new draft. The skill is looked up in
// targets, in order: the first target holding it managed gives its source.
// A skill from a local source has nobody to send to: an error.
func (s FeedbackService) Draft(ctx context.Context, targets []model.Target, in FeedbackInput) (FeedbackPreview, error) {
	if !feedback.ValidKind(in.Kind) {
		return FeedbackPreview{}, fmt.Errorf("unknown feedback kind %q: use %q or %q", in.Kind, model.FeedbackBug, model.FeedbackImprovement)
	}
	if err := feedback.CheckText(in.Title, in.Body); err != nil {
		return FeedbackPreview{}, err
	}
	marker, err := s.findSkill(ctx, targets, in.Skill)
	if err != nil {
		return FeedbackPreview{}, err
	}
	draft := model.FeedbackDraft{
		Status: model.FeedbackDraftStatus, Kind: in.Kind, Skill: in.Skill,
		Source: marker.Source, Commit: marker.Commit, SkillPath: marker.SkillPath,
		CreatedAt: s.Now(), Title: in.Title, Body: in.Body,
	}
	if _, err := s.tracker(draft); err != nil {
		return FeedbackPreview{}, err
	}
	draft.ID, err = s.Drafts.Create(ctx, feedback.DraftID(draft.CreatedAt, draft.Skill, draft.Title), draft)
	if err != nil {
		return FeedbackPreview{}, err
	}
	slog.InfoContext(ctx, "feedback draft created", "id", draft.ID, "skill", draft.Skill, "source", draft.Source.String())
	return s.preview(draft), nil
}

// Preview shows the draft id as it is now and what sending it would open.
func (s FeedbackService) Preview(ctx context.Context, id string) (FeedbackPreview, error) {
	draft, err := s.Drafts.Load(ctx, id)
	if err != nil {
		return FeedbackPreview{}, err
	}
	return s.preview(draft), nil
}

// Pending shows every stored draft still to be sent or declined, by id.
// Drafts that can't be read are reported in the error next to the ones
// that can.
func (s FeedbackService) Pending(ctx context.Context) ([]FeedbackPreview, error) {
	ids, err := s.Drafts.List(ctx)
	if err != nil {
		return nil, err
	}
	var pending []FeedbackPreview
	var failures []error
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		draft, err := s.Drafts.Load(ctx, id)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if draft.Status == model.FeedbackDraftStatus {
			pending = append(pending, s.preview(draft))
		}
	}
	return pending, errors.Join(failures...)
}

// Send opens the issue of draft id, if it is still a draft and still what
// the user confirmed (hash from Preview), and records the issue's URL in
// the draft. A draft changed since it was shown is not sent.
func (s FeedbackService) Send(ctx context.Context, id, hash string) (model.FeedbackDraft, error) {
	draft, err := s.openDraft(ctx, id)
	if err != nil {
		return draft, err
	}
	tracker, err := s.tracker(draft)
	if err != nil {
		return draft, err
	}
	// The file may have been edited by hand since Draft.
	if err := feedback.CheckText(draft.Title, draft.Body); err != nil {
		return draft, err
	}
	shown := s.preview(draft)
	if shown.Hash != hash {
		return draft, fmt.Errorf("feedback %s changed since it was confirmed: show it again", id)
	}
	url, err := tracker.Create(ctx, draft.Source, shown.Issue)
	if err != nil {
		return draft, err
	}
	draft.Status, draft.IssueURL, draft.SentAt = model.FeedbackSent, url, s.Now()
	if err := s.Drafts.Save(ctx, draft); err != nil {
		return draft, fmt.Errorf("issue %s opened, but feedback %s isn't marked sent: %w", url, id, err)
	}
	slog.InfoContext(ctx, "feedback sent", "id", id, "issue", url)
	return draft, nil
}

// Decline closes draft id without sending it; the file stays as a trace.
func (s FeedbackService) Decline(ctx context.Context, id string) (model.FeedbackDraft, error) {
	draft, err := s.openDraft(ctx, id)
	if err != nil {
		return draft, err
	}
	draft.Status = model.FeedbackDeclined
	if err := s.Drafts.Save(ctx, draft); err != nil {
		return draft, err
	}
	slog.InfoContext(ctx, "feedback declined", "id", id)
	return draft, nil
}

func (s FeedbackService) preview(draft model.FeedbackDraft) FeedbackPreview {
	issue := feedback.Compose(draft, s.Version)
	return FeedbackPreview{Draft: draft, Issue: issue, Hash: feedback.Hash(draft.Source, issue)}
}

// openDraft loads draft id, which must still be a draft.
func (s FeedbackService) openDraft(ctx context.Context, id string) (model.FeedbackDraft, error) {
	draft, err := s.Drafts.Load(ctx, id)
	if err != nil {
		return draft, err
	}
	if draft.Status != model.FeedbackDraftStatus {
		return draft, fmt.Errorf("feedback %s is already %s", id, draft.Status)
	}
	return draft, nil
}

func (s FeedbackService) findSkill(ctx context.Context, targets []model.Target, skill string) (model.ManagedState, error) {
	for _, target := range targets {
		marker, err := s.Markers.ReadMarker(ctx, target.Path, skill)
		if errors.Is(err, interfaces.ErrSkillNotManaged) {
			continue
		}
		return marker, err
	}
	paths := make([]string, len(targets))
	for i, t := range targets {
		paths[i] = t.Path
	}
	return model.ManagedState{}, fmt.Errorf("skill %q is in none of the targets %v: run sync first", skill, paths)
}

func (s FeedbackService) tracker(draft model.FeedbackDraft) (interfaces.IssueTracker, error) {
	if draft.Source.Type == "local" {
		return nil, fmt.Errorf("skill %q comes from the local source %s: edit it there by hand", draft.Skill, draft.Source.Path)
	}
	tracker, ok := s.Trackers[draft.Source.Type]
	if !ok {
		return nil, fmt.Errorf("skill %q comes from %s: no issue tracker for source type %q", draft.Skill, draft.Source.String(), draft.Source.Type)
	}
	return tracker, nil
}
