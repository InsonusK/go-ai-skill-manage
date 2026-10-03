package handler_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// markers is a fake interfaces.MarkerReader: target -> skill -> marker.
type markers map[string]map[string]model.ManagedState

func (m markers) ReadMarker(ctx context.Context, target, skill string) (model.ManagedState, error) {
	marker, ok := m[target][skill]
	if !ok {
		return marker, fmt.Errorf("%s/%s: %w", target, skill, interfaces.ErrSkillNotManaged)
	}
	return marker, nil
}

// drafts is a fake interfaces.FeedbackDrafts in memory, taking ids the
// way the real store does.
type drafts map[string]model.FeedbackDraft

func (d drafts) Create(ctx context.Context, id string, draft model.FeedbackDraft) (string, error) {
	free := id
	for n := 2; ; n++ {
		if _, taken := d[free]; !taken {
			break
		}
		free = id + "-" + strconv.Itoa(n)
	}
	draft.ID = free
	d[free] = draft
	return free, nil
}

func (d drafts) Load(ctx context.Context, id string) (model.FeedbackDraft, error) {
	draft, ok := d[id]
	if !ok {
		return draft, fmt.Errorf("no feedback %s", id)
	}
	return draft, nil
}

func (d drafts) List(ctx context.Context) ([]string, error) {
	return slices.Sorted(maps.Keys(d)), nil
}

func (d drafts) Save(ctx context.Context, draft model.FeedbackDraft) error {
	if _, ok := d[draft.ID]; !ok {
		return fmt.Errorf("no feedback %s", draft.ID)
	}
	d[draft.ID] = draft
	return nil
}

type openedIssue struct {
	Source model.SourceKey `json:"source"`
	Issue  model.NewIssue  `json:"issue"`
}

// tracker is a fake interfaces.IssueTracker recording what it opened.
type tracker struct {
	opened  *[]openedIssue
	failure *string
}

func (t tracker) Create(ctx context.Context, source model.SourceKey, issue model.NewIssue) (string, error) {
	if *t.failure != "" {
		return "", errors.New(*t.failure)
	}
	*t.opened = append(*t.opened, openedIssue{source, issue})
	return fmt.Sprintf("https://github.com/o/r/issues/%d", len(*t.opened)), nil
}

var _ interfaces.MarkerReader = markers{}
var _ interfaces.FeedbackDrafts = drafts{}
var _ interfaces.IssueTracker = tracker{}

// draftView is what scenarios compare of a draft.
type draftView struct {
	ID, Status, Kind, Skill string
	Source                  model.SourceKey
	Commit, SkillPath       string
	Title, Body, IssueURL   string
	CreatedAt, SentAt       string
}

func viewOf(d model.FeedbackDraft) draftView {
	at := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.UTC().Format(time.RFC3339)
	}
	return draftView{ID: d.ID, Status: string(d.Status), Kind: string(d.Kind), Skill: d.Skill, Source: d.Source,
		Commit: d.Commit, SkillPath: d.SkillPath, Title: d.Title, Body: d.Body, IssueURL: d.IssueURL,
		CreatedAt: at(d.CreatedAt), SentAt: at(d.SentAt)}
}

func registerFeedbackSteps(sc *godog.ScenarioContext) {
	var service handler.FeedbackService
	var managed markers
	var stored drafts
	var opened []openedIssue
	var trackerFailure string
	var targets []model.Target
	var now time.Time
	var preview handler.FeedbackPreview
	var failure error

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		managed, stored, opened, trackerFailure, targets, failure = markers{}, drafts{}, nil, "", nil, nil
		preview = handler.FeedbackPreview{}
		now = time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
		service = handler.FeedbackService{
			Markers: managed, Drafts: stored,
			Trackers: map[string]interfaces.IssueTracker{"github": tracker{opened: &opened, failure: &trackerFailure}},
			Now:      func() time.Time { return now }, Version: "1.2.0",
		}
		return ctx, nil
	})

	// the managed skills are: a table of target, skill and the marker's
	// source type, path, tree, commit and skill_path.
	sc.Step(`^the managed skills are$`, func(ctx context.Context, t *godog.Table) error {
		for _, row := range t.Rows[1:] {
			c := row.Cells
			if managed[c[0].Value] == nil {
				managed[c[0].Value] = map[string]model.ManagedState{}
			}
			managed[c[0].Value][c[1].Value] = model.ManagedState{
				Source: model.SourceKey{Type: c[2].Value, Path: c[3].Value, Tree: c[4].Value},
				Commit: c[5].Value, SkillPath: c[6].Value,
			}
		}
		return nil
	})
	sc.Step(`^the feedback targets are "([^"]*)"$`, func(ctx context.Context, paths string) error {
		for _, p := range list(paths) {
			targets = append(targets, model.Target{Name: p, Path: p})
		}
		return nil
	})
	sc.Step(`^it is "([^"]*)"$`, func(ctx context.Context, at string) (err error) {
		now, err = time.Parse(time.RFC3339, at)
		return err
	})
	sc.Step(`^I draft an? "([^"]*)" feedback on "([^"]*)" titled "([^"]*)" with body "([^"]*)"$`, func(ctx context.Context, kind, skill, title, body string) error {
		body = strings.ReplaceAll(body, `\n`, "\n")
		preview, failure = service.Draft(ctx, targets, handler.FeedbackInput{Skill: skill, Kind: model.FeedbackKind(kind), Title: title, Body: body})
		testsupport.Log("draft=%+v error=%v", preview.Draft, failure)
		return nil
	})
	sc.Step(`^I preview "([^"]*)"$`, func(ctx context.Context, id string) error {
		preview, failure = service.Preview(ctx, id)
		testsupport.Log("preview=%+v error=%v", preview, failure)
		return nil
	})
	var pending []handler.FeedbackPreview
	sc.Step(`^I ask for the pending feedback$`, func(ctx context.Context) error {
		pending, failure = service.Pending(ctx)
		testsupport.Log("pending=%+v error=%v", pending, failure)
		return nil
	})
	sc.Step(`^the pending feedback is "([^"]*)"$`, func(ctx context.Context, want string) error {
		var ids []string
		for _, p := range pending {
			if p.Hash == "" {
				return fmt.Errorf("pending %s has no hash", p.Draft.ID)
			}
			ids = append(ids, p.Draft.ID)
		}
		return testsupport.Equal(strings.Join(ids, ","), want)
	})
	sc.Step(`^the user edits the body of "([^"]*)" to "([^"]*)"$`, func(ctx context.Context, id, body string) error {
		d := stored[id]
		d.Body = body
		stored[id] = d
		return nil
	})
	sc.Step(`^I send "([^"]*)" with the shown hash$`, func(ctx context.Context, id string) error {
		_, failure = service.Send(ctx, id, preview.Hash)
		testsupport.Log("send %s hash=%s error=%v", id, preview.Hash, failure)
		return nil
	})
	sc.Step(`^I send "([^"]*)" with hash "([^"]*)"$`, func(ctx context.Context, id, hash string) error {
		_, failure = service.Send(ctx, id, hash)
		testsupport.Log("send %s hash=%s error=%v", id, hash, failure)
		return nil
	})
	sc.Step(`^I decline "([^"]*)"$`, func(ctx context.Context, id string) error {
		_, failure = service.Decline(ctx, id)
		testsupport.Log("decline %s error=%v", id, failure)
		return nil
	})
	sc.Step(`^the tracker fails with "([^"]*)"$`, func(ctx context.Context, msg string) error {
		trackerFailure = msg
		return nil
	})
	sc.Step(`^the feedback succeeds$`, func(ctx context.Context) error { return failure })
	sc.Step(`^the feedback fails with "(.*)"$`, func(ctx context.Context, want string) error {
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %q", failure, want)
		}
		return nil
	})
	sc.Step(`^the drafts are "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(slices.Sorted(maps.Keys(stored)), list(want))
	})
	sc.Step(`^the draft "([^"]*)" is$`, func(ctx context.Context, id string, d *godog.DocString) error {
		draft, ok := stored[id]
		if !ok {
			return fmt.Errorf("no draft %s", id)
		}
		return testsupport.JSON(viewOf(draft), d)
	})
	sc.Step(`^the shown issue is$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(preview.Issue, d) })
	sc.Step(`^the tracker opened$`, func(ctx context.Context, d *godog.DocString) error {
		if opened == nil {
			opened = []openedIssue{}
		}
		return testsupport.JSON(opened, d)
	})
}
