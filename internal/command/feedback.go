package command

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// FeedbackDraftsDir is where feedback drafts lie, from the project's root
// (the config file's folder). They are meant to be committed.
const FeedbackDraftsDir = ".ai-skills/feedback"

// feedback runs the feedback command. Only draft, show and decline work
// without a terminal; send asks the user and so needs one: an agent can
// draft, but only the user sends.
func (a App) feedback(ctx context.Context, opts Options, req model.Request) int {
	f := opts.Feedback
	service, dir := a.feedbackService(req)
	id := draftID(f.ID)
	switch f.Action {
	case "draft":
		body := f.Body
		if f.BodyFile != "" {
			raw, err := a.readBody(f.BodyFile)
			if err != nil {
				fmt.Fprintln(a.Err, err)
				return 1
			}
			body = raw
		}
		preview, err := service.Draft(ctx, req.Targets, handler.FeedbackInput{Skill: f.Skill, Kind: model.FeedbackKind(f.Kind), Title: f.Title, Body: body})
		if err != nil {
			fmt.Fprintln(a.Err, err)
			return 1
		}
		fmt.Fprintf(a.Out, "Feedback draft: %s\n\n", filepath.Join(dir, preview.Draft.ID+".md"))
		printIssue(a.Out, preview)
		fmt.Fprintf(a.Out, "\nNothing is sent yet. Review the draft (edit the file if needed); then the user sends it:\n  %s\n", sendCommand(opts, preview.Draft.ID))
		return 0
	case "show":
		preview, err := service.Preview(ctx, id)
		if err != nil {
			fmt.Fprintln(a.Err, err)
			return 1
		}
		fmt.Fprintf(a.Out, "Feedback %s: %s\n", id, preview.Draft.Status)
		if preview.Draft.IssueURL != "" {
			fmt.Fprintf(a.Out, "Issue: %s\n", preview.Draft.IssueURL)
		}
		fmt.Fprintln(a.Out)
		printIssue(a.Out, preview)
		return 0
	case "decline":
		if _, err := service.Decline(ctx, id); err != nil {
			fmt.Fprintln(a.Err, err)
			return 1
		}
		fmt.Fprintf(a.Out, "Feedback %s declined; the draft stays as a trace\n", id)
		return 0
	default:
		return a.sendFeedback(ctx, opts, service, id)
	}
}

// feedbackService is the feedback service of req's project and its drafts
// folder -- the same for the CLI and the MCP server.
func (a App) feedbackService(req model.Request) (handler.FeedbackService, string) {
	now := a.Now
	if now == nil {
		now = time.Now
	}
	dir := filepath.Join(req.Base, FeedbackDraftsDir)
	return handler.FeedbackService{Markers: a.Markers, Drafts: a.Drafts(dir), Trackers: a.Trackers, Now: now, Version: a.Version}, dir
}

// sendFeedback shows what would be sent and sends it only on the user's
// "y" typed in a terminal.
func (a App) sendFeedback(ctx context.Context, opts Options, service handler.FeedbackService, id string) int {
	if a.IsTerminal == nil || !a.IsTerminal() {
		fmt.Fprintf(a.Err, "feedback send asks the user to confirm and needs a terminal: ask the user to run it:\n  %s\n", sendCommand(opts, id))
		return 1
	}
	preview, err := service.Preview(ctx, id)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	if preview.Draft.Status != model.FeedbackDraftStatus {
		fmt.Fprintf(a.Err, "feedback %s is already %s\n", id, preview.Draft.Status)
		return 1
	}
	printIssue(a.Out, preview)
	fmt.Fprint(a.Out, "\nOpen this issue? It is public if the repository is. [y/N] ")
	answer, err := bufio.NewReader(a.In).ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	if reply := strings.ToLower(strings.TrimSpace(answer)); reply != "y" && reply != "yes" {
		fmt.Fprintln(a.Out, "Not sent.")
		return 0
	}
	draft, err := service.Send(ctx, id, preview.Hash)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	fmt.Fprintf(a.Out, "Sent: %s\n", draft.IssueURL)
	return 0
}

func (a App) readBody(file string) (string, error) {
	if file == "-" {
		raw, err := io.ReadAll(a.In)
		return string(raw), err
	}
	raw, err := a.ReadFile(file)
	return string(raw), err
}

// printIssue prints where the issue goes and what it says.
//
// Пример:
//
//	Repository: github https://github.com/o/r
//	Labels: bug
//	Title: Broken anchor
//
//	The anchor #x is missing.
//	...
func printIssue(out io.Writer, preview handler.FeedbackPreview) {
	source := preview.Draft.Source
	fmt.Fprintf(out, "Repository: %s %s\n", source.Type, source.Path)
	fmt.Fprintf(out, "Labels: %s\n", strings.Join(preview.Issue.Labels, ", "))
	fmt.Fprintf(out, "Title: %s\n\n", preview.Issue.Title)
	fmt.Fprint(out, preview.Issue.Body)
}

// draftID accepts the id, its file name or its path.
func draftID(arg string) string { return strings.TrimSuffix(filepath.Base(arg), ".md") }

func sendCommand(opts Options, id string) string {
	cmd := "ai-skill-manager feedback send " + id
	if opts.Config != "" {
		cmd += " -c " + opts.Config
	}
	return cmd
}
