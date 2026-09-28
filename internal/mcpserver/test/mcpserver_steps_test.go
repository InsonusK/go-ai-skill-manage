package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/internal/mcpserver"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// markers is a fake interfaces.MarkerReader of one target: skill -> marker.
type markers map[string]model.ManagedState

func (m markers) ReadMarker(ctx context.Context, target, skill string) (model.ManagedState, error) {
	marker, ok := m[skill]
	if !ok {
		return marker, fmt.Errorf("%s: %w", skill, interfaces.ErrSkillNotManaged)
	}
	return marker, nil
}

// tracker is a fake interfaces.IssueTracker recording the titles it opened.
type tracker struct{ titles *[]string }

func (t tracker) Create(ctx context.Context, source model.SourceKey, issue model.NewIssue) (string, error) {
	*t.titles = append(*t.titles, issue.Title)
	return fmt.Sprintf("https://github.com/o/r/issues/%d", len(*t.titles)), nil
}

func initialize(sc *godog.ScenarioContext) {
	var dir, protocol, lastID string
	var managed markers
	var opened, dialogs []string
	var canAsk, urlOnly, editDuringDialog bool
	var action string
	var send any
	var session *mcp.ClientSession
	var result *mcp.CallToolResult
	drafts := func() filesystem.FeedbackDrafts { return filesystem.FeedbackDrafts{Dir: dir} }

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		var err error
		dir, err = os.MkdirTemp("", "aism-mcp-test-")
		protocol, lastID, managed, opened, dialogs = "", "", markers{}, []string{}, []string{}
		canAsk, urlOnly, editDuringDialog, action, send, session, result = false, false, false, "", nil, nil, nil
		return ctx, err
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if session != nil {
			_ = session.Close()
		}
		return ctx, os.RemoveAll(dir)
	})

	connect := func(ctx context.Context) error {
		if session != nil {
			return nil
		}
		open := func(context.Context) (mcpserver.Project, error) {
			service := handler.FeedbackService{
				Markers: managed, Drafts: drafts(), Trackers: map[string]interfaces.IssueTracker{"github": tracker{titles: &opened}},
				Now: func() time.Time { return time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC) }, Version: "1.2.0",
			}
			return mcpserver.Project{Service: service, Targets: []model.Target{{Name: "claude", Path: "/p/.claude/skills"}}, DraftsDir: dir}, nil
		}
		options := &mcp.ClientOptions{}
		if canAsk {
			options.ElicitationHandler = func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
				dialogs = append(dialogs, req.Params.Message)
				if editDuringDialog {
					d, err := drafts().Load(ctx, lastID)
					if err != nil {
						return nil, err
					}
					d.Body = "edited while the dialog was open"
					if err := drafts().Save(ctx, d); err != nil {
						return nil, err
					}
				}
				answer := &mcp.ElicitResult{Action: action}
				if action == "accept" {
					answer.Content = map[string]any{"send": send}
				}
				return answer, nil
			}
		}
		if urlOnly {
			// Answers every dialog "accept": a form must never be asked for.
			options.ElicitationHandler = func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
				dialogs = append(dialogs, req.Params.Message)
				return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"send": true}}, nil
			}
			options.Capabilities = &mcp.ClientCapabilities{Elicitation: &mcp.ElicitationCapabilities{URL: &mcp.URLElicitationCapabilities{}}}
		}
		serverSide, clientSide := mcp.NewInMemoryTransports()
		if _, err := mcpserver.NewServer(open, "1.2.0").Connect(ctx, serverSide, nil); err != nil {
			return err
		}
		var err error
		session, err = mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, options).Connect(ctx, clientSide, &mcp.ClientSessionOptions{ProtocolVersion: protocol})
		return err
	}
	call := func(ctx context.Context, tool string, args map[string]any) error {
		if err := connect(ctx); err != nil {
			return err
		}
		var err error
		result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(result)
		testsupport.Log("%s -> %s", tool, raw)
		return nil
	}

	sc.Step(`^the client speaks protocol "([^"]*)"$`, func(ctx context.Context, v string) error { protocol = v; return nil })
	sc.Step(`^the skill "([^"]*)" comes from "([^"]*)" "([^"]*)"$`, func(ctx context.Context, skill, kind, path string) error {
		managed[skill] = model.ManagedState{Source: model.SourceKey{Type: kind, Path: path}, Commit: "c0ffee", SkillPath: "a/" + skill}
		return nil
	})
	sc.Step(`^the client shows only links, no forms$`, func(ctx context.Context) error { urlOnly = true; return nil })
	sc.Step(`^the client can't show dialogs$`, func(ctx context.Context) error { canAsk = false; return nil })
	sc.Step(`^the user answers the dialog "([^"]*)"( ticking send| leaving send unticked)?$`, func(ctx context.Context, a, tick string) error {
		canAsk, action = true, a
		switch tick {
		case " ticking send":
			send = true
		case " leaving send unticked":
			send = false
		}
		return nil
	})
	sc.Step(`^the user edits the draft while the dialog is open$`, func(ctx context.Context) error { editDuringDialog = true; return nil })
	sc.Step(`^the agent drafts a "([^"]*)" on "([^"]*)" titled "([^"]*)" with body "([^"]*)"$`, func(ctx context.Context, kind, skill, title, body string) error {
		if err := call(ctx, "feedback_draft", map[string]any{"skill": skill, "kind": kind, "title": title, "body": body}); err != nil {
			return err
		}
		if out, ok := result.StructuredContent.(map[string]any); ok {
			lastID, _ = out["id"].(string)
		}
		return nil
	})
	sc.Step(`^the agent submits the draft$`, func(ctx context.Context) error {
		return call(ctx, "feedback_submit", map[string]any{"id": lastID})
	})
	// the tool result has: the listed fields of the structured result;
	// DRAFTS is the drafts folder.
	sc.Step(`^the tool result has$`, func(ctx context.Context, d *godog.DocString) error {
		if result.IsError {
			return fmt.Errorf("tool error: %s", text(result))
		}
		var want map[string]any
		if err := json.Unmarshal([]byte(strings.ReplaceAll(d.Content, "DRAFTS", dir)), &want); err != nil {
			return err
		}
		got, _ := result.StructuredContent.(map[string]any)
		picked := map[string]any{}
		for k := range want {
			picked[k] = got[k]
		}
		return testsupport.Equal(picked, want)
	})
	sc.Step(`^the tool fails with "(.*)"$`, func(ctx context.Context, want string) error {
		if !result.IsError || !strings.Contains(text(result), want) {
			return fmt.Errorf("isError=%t text=%q want %q", result.IsError, text(result), want)
		}
		return nil
	})
	sc.Step(`^the user saw the dialog$`, func(ctx context.Context, d *godog.DocString) error {
		return testsupport.Equal(dialogs, []string{d.Content})
	})
	sc.Step(`^the user saw no dialog$`, func(ctx context.Context) error { return testsupport.Equal(dialogs, []string{}) })
	sc.Step(`^the opened issues are "([^"]*)"$`, func(ctx context.Context, titles string) error {
		want := []string{}
		if titles != "" {
			want = strings.Split(titles, ",")
		}
		return testsupport.Equal(opened, want)
	})
	sc.Step(`^the draft is "([^"]*)"$`, func(ctx context.Context, status string) error {
		d, err := drafts().Load(ctx, lastID)
		if err != nil {
			return err
		}
		return testsupport.Equal(string(d.Status), status)
	})
	sc.Step(`^the draft file lies in the drafts folder$`, func(ctx context.Context) error {
		out, _ := result.StructuredContent.(map[string]any)
		_, err := os.Stat(fmt.Sprint(out["file"]))
		return err
	})
}

func text(r *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range r.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			b.WriteString(t.Text)
		}
	}
	return b.String()
}
