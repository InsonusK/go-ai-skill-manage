// Package mcpserver is the MCP inbound adapter: it lets an agent draft a
// feedback about a skill and ask the user, through the client's
// elicitation dialog, to send it (handler.FeedbackService). The agent has
// no tool that sends without the user.
package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Project is the feedback service of the project and what it needs: the
// targets to look the skill up in and the drafts folder.
type Project struct {
	Service   handler.FeedbackService
	Targets   []model.Target
	DraftsDir string
}

// Open reads the project's configuration anew for every call, so a
// changed config applies without restarting the server.
type Open func(context.Context) (Project, error)

// DraftInput is what the agent writes about a skill.
type DraftInput struct {
	Skill string `json:"skill" jsonschema:"the skill's name: its folder in .claude/skills or .agents/skills"`
	Kind  string `json:"kind" jsonschema:"bug or improvement"`
	Title string `json:"title" jsonschema:"issue title, one line"`
	Body  string `json:"body" jsonschema:"issue text in markdown: what is wrong or what to improve, and how to reproduce; no code or data of the user's project"`
}

// DraftOutput is the stored draft and the issue it would open.
type DraftOutput struct {
	ID         string   `json:"id"`
	File       string   `json:"file"`
	Repository string   `json:"repository"`
	Labels     []string `json:"labels"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Next       string   `json:"next"`
}

type SubmitInput struct {
	ID string `json:"id" jsonschema:"the draft id feedback_draft returned"`
}

// SubmitOutput tells whether the issue was opened.
type SubmitOutput struct {
	Sent     bool   `json:"sent"`
	IssueURL string `json:"issue_url,omitempty"`
	Message  string `json:"message"`
}

// confirmRequest names the confirmation among the input requests;
// confirmField is its form's only field: the user ticks it to send.
const (
	confirmRequest = "confirm"
	confirmField   = "send"
)

// NewServer makes the MCP server with the feedback tools.
func NewServer(open Open, version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "ai-skill-manager", Title: "AI Skill Manager", Version: version}, &mcp.ServerOptions{
		Instructions: "Report a bug or suggest an improvement to a skill you use, as an issue in the repository the skill was synced from. " +
			"Call feedback_draft, then feedback_submit: the user reads the issue and decides whether to send it. Never put the user's code or data in the issue.",
	})
	no, yes := false, true
	mcp.AddTool(server, &mcp.Tool{
		Name: "feedback_draft",
		Description: "Write a feedback about a skill (bug or improvement) as a draft file in the project. Sends nothing. " +
			"The skill's source repository is found from the skill's folder. Show the user the returned file, then call feedback_submit.",
		Annotations: &mcp.ToolAnnotations{Title: "Draft skill feedback", DestructiveHint: &no, OpenWorldHint: &no},
	}, draftTool(open))
	mcp.AddTool(server, &mcp.Tool{
		Name: "feedback_submit",
		Description: "Ask the user to confirm a feedback draft and, only if the user confirms, open it as an issue in the skill's source repository. " +
			"The user sees the full issue in a dialog. If the client can't show one, the user sends it from a terminal.",
		Annotations: &mcp.ToolAnnotations{Title: "Send skill feedback after the user confirms", DestructiveHint: &no, OpenWorldHint: &yes},
	}, submitTool(open))
	return server
}

func draftTool(open Open) mcp.ToolHandlerFor[DraftInput, DraftOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in DraftInput) (*mcp.CallToolResult, DraftOutput, error) {
		project, err := open(ctx)
		if err != nil {
			return nil, DraftOutput{}, err
		}
		preview, err := project.Service.Draft(ctx, project.Targets, handler.FeedbackInput{Skill: in.Skill, Kind: model.FeedbackKind(in.Kind), Title: in.Title, Body: in.Body})
		if err != nil {
			return nil, DraftOutput{}, err
		}
		id := preview.Draft.ID
		return nil, DraftOutput{
			ID: id, File: filepath.Join(project.DraftsDir, id+".md"),
			Repository: repository(preview.Draft.Source), Labels: preview.Issue.Labels, Title: preview.Issue.Title, Body: preview.Issue.Body,
			Next: "Nothing is sent. Tell the user the draft file to review (they may edit it), then call feedback_submit with id " + id + ".",
		}, nil
	}
}

// submitTool asks the user in two rounds (multi round-trip requests,
// SEP-2322; for clients before protocol 2026-07-28 the SDK makes the same
// rounds with a server-initiated elicitation): the first call returns the
// confirmation form with the shown issue's hash as the request state; the
// retry carries the user's answer and sends only that hash's issue.
func submitTool(open Open) mcp.ToolHandlerFor[SubmitInput, SubmitOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in SubmitInput) (*mcp.CallToolResult, SubmitOutput, error) {
		id := strings.TrimSuffix(filepath.Base(in.ID), ".md")
		project, err := open(ctx)
		if err != nil {
			return nil, SubmitOutput{}, err
		}
		if answer, ok := req.Params.InputResponses[confirmRequest].(*mcp.ElicitResult); ok {
			if answer.Action != "accept" || answer.Content[confirmField] != true {
				return nil, SubmitOutput{Message: "Not sent: the user didn't confirm. The draft stays; don't ask again unless the user wants to."}, nil
			}
			draft, err := project.Service.Send(ctx, id, req.Params.RequestState)
			if err != nil {
				return nil, SubmitOutput{}, err
			}
			return nil, SubmitOutput{Sent: true, IssueURL: draft.IssueURL, Message: "Sent: " + draft.IssueURL}, nil
		}
		preview, err := project.Service.Preview(ctx, id)
		if err != nil {
			return nil, SubmitOutput{}, err
		}
		if preview.Draft.Status != model.FeedbackDraftStatus {
			return nil, SubmitOutput{}, fmt.Errorf("feedback %s is already %s", id, preview.Draft.Status)
		}
		if !canAsk(req.ClientCapabilities()) {
			return nil, SubmitOutput{Message: "Not sent: this client can't ask the user to confirm. Ask the user to review " +
				filepath.Join(project.DraftsDir, id+".md") + " and send it from their terminal: ai-skill-manager feedback send " + id}, nil
		}
		return &mcp.CallToolResult{
			InputRequests: mcp.InputRequestMap{confirmRequest: &mcp.ElicitParams{
				Mode:    "form",
				Message: confirmation(preview),
				RequestedSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						confirmField: map[string]any{"type": "boolean", "title": "Open this issue", "description": "Public if the repository is", "default": false},
					},
					"required": []string{confirmField},
				},
			}},
			// Send checks it against the draft as it is then: a draft
			// changed after it was shown isn't sent.
			RequestState: preview.Hash,
		}, SubmitOutput{}, nil
	}
}

// canAsk tells whether the client shows elicitation forms.
func canAsk(caps *mcp.ClientCapabilities) bool {
	if caps == nil || caps.Elicitation == nil {
		return false
	}
	// Neither mode named: form, for older clients.
	return caps.Elicitation.Form != nil || caps.Elicitation.URL == nil
}

// confirmation is the dialog's text: where the issue goes and all of it.
func confirmation(preview handler.FeedbackPreview) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Open an issue in %s?\n\n", repository(preview.Draft.Source))
	fmt.Fprintf(&b, "Labels: %s\nTitle: %s\n\n%s", strings.Join(preview.Issue.Labels, ", "), preview.Issue.Title, preview.Issue.Body)
	return b.String()
}

func repository(source model.SourceKey) string { return source.Type + " " + source.Path }
