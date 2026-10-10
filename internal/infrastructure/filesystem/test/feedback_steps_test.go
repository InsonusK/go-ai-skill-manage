package filesystem_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

const draftsDir = ".ai-skills/feedback"

// feedbackSteps: markers are read from, and drafts kept in, the folder of
// "an empty target" (*dir); drafts lie in its draftsDir.
func feedbackSteps(sc *godog.ScenarioContext, dir *string) {
	var marker model.ManagedState
	var draft model.FeedbackDraft
	var id string
	var failure error
	drafts := func() filesystem.FeedbackDrafts {
		return filesystem.FeedbackDrafts{Dir: filepath.Join(*dir, draftsDir)}
	}

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		marker, draft, id, failure = model.ManagedState{}, model.FeedbackDraft{}, "", nil
		return ctx, nil
	})

	sc.Step(`^the marker of "([^"]*)" is$`, func(ctx context.Context, skill string, d *godog.DocString) error {
		p := filepath.Join(*dir, skill, model.Marker)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte(d.Content), 0o644)
	})
	sc.Step(`^I read the marker of "([^"]*)"( in a target folder that doesn't exist)?$`, func(ctx context.Context, skill, missing string) error {
		target := *dir
		if missing != "" {
			target = filepath.Join(*dir, "missing")
		}
		marker, failure = (filesystem.Store{}).ReadMarker(ctx, target, skill)
		testsupport.Log("marker=%+v error=%v", marker, failure)
		return nil
	})
	sc.Step(`^the marker read is$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		return testsupport.JSON(marker, d)
	})
	sc.Step(`^the skill is not managed$`, func(ctx context.Context) error {
		return testsupport.Equal(fmt.Sprint(failure != nil && errors.Is(failure, interfaces.ErrSkillNotManaged)), "true")
	})
	sc.Step(`^reading the marker fails with "(.*)"$`, func(ctx context.Context, want string) error {
		return errorContains(failure, want)
	})

	sc.Step(`^I create the draft "([^"]*)" of a "([^"]*)" on "([^"]*)" titled "([^"]*)" with body "([^"]*)"$`, func(ctx context.Context, want, kind, skill, title, body string) error {
		d := model.FeedbackDraft{
			Status: model.FeedbackDraftStatus, Kind: model.FeedbackKind(kind), Skill: skill,
			Source: model.SourceKey{Type: "github", Path: "https://github.com/o/r", Tree: "main"}, Commit: "c0ffee", SkillPath: "a/" + skill,
			CreatedAt: time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC), Title: title, Body: strings.ReplaceAll(body, `\n`, "\n"),
		}
		id, failure = drafts().Create(ctx, want, d)
		testsupport.Log("created id=%s error=%v", id, failure)
		return nil
	})
	sc.Step(`^the created id is "([^"]*)"$`, func(ctx context.Context, want string) error {
		if failure != nil {
			return failure
		}
		return testsupport.Equal(id, want)
	})
	var listed []string
	sc.Step(`^I list the drafts$`, func(ctx context.Context) error {
		listed, failure = drafts().List(ctx)
		testsupport.Log("listed=%v error=%v", listed, failure)
		return failure
	})
	sc.Step(`^the drafts listed are "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(listed, ","), want)
	})
	sc.Step(`^I load the draft "([^"]*)"$`, func(ctx context.Context, want string) error {
		draft, failure = drafts().Load(ctx, want)
		testsupport.Log("draft=%+v error=%v", draft, failure)
		return nil
	})
	sc.Step(`^I mark the loaded draft sent as "([^"]*)" at "([^"]*)" and save it$`, func(ctx context.Context, url, at string) error {
		sent, err := time.Parse(time.RFC3339, at)
		if err != nil {
			return err
		}
		draft.Status, draft.IssueURL, draft.SentAt = model.FeedbackSent, url, sent
		failure = drafts().Save(ctx, draft)
		return nil
	})
	sc.Step(`^I save a draft "([^"]*)"$`, func(ctx context.Context, want string) error {
		failure = drafts().Save(ctx, model.FeedbackDraft{ID: want, Status: model.FeedbackDraftStatus, Title: "t", Body: "b"})
		return nil
	})
	sc.Step(`^the draft file "([^"]*)" is$`, func(ctx context.Context, name string, d *godog.DocString) error {
		raw, err := os.ReadFile(filepath.Join(*dir, draftsDir, name))
		if err != nil {
			return err
		}
		return testsupport.Equal(string(raw), d.Content+"\n")
	})
	sc.Step(`^the draft file "([^"]*)" contains$`, func(ctx context.Context, name string, d *godog.DocString) error {
		p := filepath.Join(*dir, draftsDir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, []byte(strings.ReplaceAll(d.Content, `\r`, "\r")+"\n"), 0o644)
	})
	// the loaded draft is: {ID, Status, Kind, Skill, Source, Commit,
	// SkillPath, Title, Body, IssueURL, CreatedAt, SentAt}, times RFC 3339.
	sc.Step(`^the loaded draft is$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		at := func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.UTC().Format(time.RFC3339)
		}
		return testsupport.JSON(map[string]any{
			"ID": draft.ID, "Status": draft.Status, "Kind": draft.Kind, "Skill": draft.Skill, "Source": draft.Source,
			"Commit": draft.Commit, "SkillPath": draft.SkillPath, "Title": draft.Title, "Body": draft.Body,
			"IssueURL": draft.IssueURL, "CreatedAt": at(draft.CreatedAt), "SentAt": at(draft.SentAt),
		}, d)
	})
	sc.Step(`^the drafts fail with "(.*)"$`, func(ctx context.Context, want string) error { return errorContains(failure, want) })
}

func errorContains(err error, want string) error {
	testsupport.Log("error=%v", err)
	if err == nil || !strings.Contains(err.Error(), want) {
		return fmt.Errorf("error=%v want %q", err, want)
	}
	return nil
}
