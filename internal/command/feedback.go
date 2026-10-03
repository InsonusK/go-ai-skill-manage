package command

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/command/common"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Feedback is the feedback command: report a bug or suggest an improvement
// to a skill's source. Action is one of feedbackActions; draft takes
// Skill, Kind, Title and Body or BodyFile ("-" for stdin); the others take
// ID; send also takes sendAll in place of it.
type Feedback struct {
	Action, ID                         string
	Source                             common.Source
	Skill, Kind, Title, Body, BodyFile string
}

// sendAll in place of the draft id: every draft not yet sent or declined.
// No draft has this id: an id starts with the date.
const sendAll = "all"

// feedbackActions: name, usage after "aism feedback", summary.
var feedbackActions = [][3]string{
	{"draft", "draft --skill NAME --kind bug|improvement --title TEXT\n      (--body TEXT | --body-file FILE|-) [-c FILE]",
		"Write the draft to .ai-skills/feedback/ next to the config;\nnothing is sent"},
	{"show", "show ID [-c FILE]", "Show the draft and the issue it would open"},
	{"send", "send ID|all [-c FILE]", "Show it, ask for confirmation, open the issue\n(asks in a terminal only: the user runs it);\nall: every draft not yet sent or declined, asking for each"},
	{"decline", "decline ID [-c FILE]", "Close the draft without sending; the file stays"},
}

func (f *Feedback) Name() string { return "feedback" }
func (f *Feedback) Summary() string {
	return "Report a bug or suggest an improvement to a skill's source\n(a GitHub issue), sent only after the user confirms it"
}

func (f *Feedback) flags() []common.Flag {
	flags := []common.Flag{f.Source.ConfigFlag()}
	if f.Action == "draft" {
		flags = append(flags,
			common.String(&f.Skill, "NAME", "The skill, as its folder in a target", "--skill"),
			common.String(&f.Kind, "KIND", "bug or improvement", "--kind"),
			common.String(&f.Title, "TEXT", "The issue's title, one line", "--title"),
			common.String(&f.Body, "TEXT", "The issue's text", "--body"),
			common.String(&f.BodyFile, "FILE", "The issue's text from FILE; - is stdin", "--body-file"),
		)
	}
	return flags
}

// Parse: the action comes first, then its flags and, but for draft, the
// draft's id.
func (f *Feedback) Parse(args []string, global []common.Flag) error {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if !slices.ContainsFunc(feedbackActions, func(a [3]string) bool { return a[0] == args[0] }) {
			return fmt.Errorf("unknown feedback action %q: use %s", args[0], strings.Join(feedbackActionNames(), ", "))
		}
		f.Action, args = args[0], args[1:]
	}
	rest, err := common.ParseFlags(args, append(f.flags(), global...), strings.TrimSpace("feedback "+f.Action))
	if err != nil {
		return err
	}
	if f.Action == "" {
		return fmt.Errorf("feedback needs an action: %s", strings.Join(feedbackActionNames(), ", "))
	}
	if f.Action != "draft" && len(rest) > 0 {
		f.ID, rest = rest[0], rest[1:]
	}
	switch {
	case len(rest) > 0:
		return fmt.Errorf("unexpected argument %q", rest[0])
	case f.Action != "draft" && f.ID == "":
		return fmt.Errorf("feedback %s needs the draft id", f.Action)
	case f.Action == "draft" && (f.Skill == "" || f.Kind == "" || f.Title == ""):
		return fmt.Errorf("feedback draft needs --skill, --kind and --title")
	case f.Action == "draft" && (f.Body == "") == (f.BodyFile == ""):
		return fmt.Errorf("feedback draft needs either --body or --body-file")
	}
	return nil
}

func feedbackActionNames() []string {
	var names []string
	for _, a := range feedbackActions {
		names = append(names, a[0])
	}
	return names
}

func (f *Feedback) Help(global []common.Flag) string {
	const about = "ID is the draft's name, its file or its path. The config's folder is the\n" +
		"project's root: the targets and .ai-skills/feedback/ are found from it."
	i := slices.IndexFunc(feedbackActions, func(a [3]string) bool { return a[0] == f.Action })
	if i < 0 {
		h := common.Help{
			Usage:        []string{"aism feedback <action> [options]"},
			Description:  "Report a bug or suggest an improvement to a skill's source as a GitHub issue.\nAn agent may draft; only the user sends, after reading what is sent.\n\n" + about,
			EntriesTitle: "Actions",
			Flags:        f.flags(),
			Global:       global,
			Footer:       "Run \"aism feedback <action> --help\" for an action's options.",
		}
		for _, a := range feedbackActions {
			h.Entries = append(h.Entries, [2]string{a[0], a[2]})
		}
		return h.String()
	}
	action := feedbackActions[i]
	description := action[2] + ".\n\n" + about
	if f.Action == "draft" {
		description = action[2] + ". The skill's source is found from its\n" +
			".ai-skills-managed marker in the first target that has the skill."
	}
	return common.Help{
		Usage:       []string{"aism feedback " + strings.ReplaceAll(action[1], "\n", "\n       ")},
		Description: description,
		Flags:       f.flags(),
		Global:      global,
	}.String()
}

// Run runs the action. Only draft, show and decline work without a
// terminal; send asks the user and so needs one: an agent can draft, but
// only the user sends.
func (f *Feedback) Run(ctx context.Context, app *common.App, cwd string) int {
	req, ok := app.LoadRequest(ctx, f.Source, config.Overrides{}, cwd)
	if !ok {
		return 1
	}
	service, dir := app.FeedbackService(req)
	id := draftID(f.ID)
	switch f.Action {
	case "draft":
		body := f.Body
		if f.BodyFile != "" {
			raw, err := readBody(app, f.BodyFile)
			if err != nil {
				fmt.Fprintln(app.Err, err)
				return 1
			}
			body = raw
		}
		preview, err := service.Draft(ctx, req.Targets, handler.FeedbackInput{Skill: f.Skill, Kind: model.FeedbackKind(f.Kind), Title: f.Title, Body: body})
		if err != nil {
			fmt.Fprintln(app.Err, err)
			return 1
		}
		fmt.Fprintf(app.Out, "Feedback draft: %s\n\n", filepath.Join(dir, preview.Draft.ID+".md"))
		printIssue(app.Out, preview)
		fmt.Fprintf(app.Out, "\nNothing is sent yet. Review the draft (edit the file if needed); then the user sends it:\n  %s\n", f.sendCommand(preview.Draft.ID))
		return 0
	case "show":
		preview, err := service.Preview(ctx, id)
		if err != nil {
			fmt.Fprintln(app.Err, err)
			return 1
		}
		fmt.Fprintf(app.Out, "Feedback %s: %s\n", id, preview.Draft.Status)
		if preview.Draft.IssueURL != "" {
			fmt.Fprintf(app.Out, "Issue: %s\n", preview.Draft.IssueURL)
		}
		fmt.Fprintln(app.Out)
		printIssue(app.Out, preview)
		return 0
	case "decline":
		if _, err := service.Decline(ctx, id); err != nil {
			fmt.Fprintln(app.Err, err)
			return 1
		}
		fmt.Fprintf(app.Out, "Feedback %s declined; the draft stays as a trace\n", id)
		return 0
	default:
		return f.send(ctx, app, service, id)
	}
}

// send shows what would be sent and sends it only on the user's
// "y" typed in a terminal.
func (f *Feedback) send(ctx context.Context, app *common.App, service handler.FeedbackService, id string) int {
	if app.IsTerminal == nil || !app.IsTerminal() {
		fmt.Fprintf(app.Err, "feedback send asks the user to confirm and needs a terminal: ask the user to run it:\n  %s\n", f.sendCommand(id))
		return 1
	}
	// One reader for all the answers: it reads ahead.
	answers := bufio.NewReader(app.In)
	if id == sendAll {
		return sendPending(ctx, app, service, answers)
	}
	preview, err := service.Preview(ctx, id)
	if err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	if preview.Draft.Status != model.FeedbackDraftStatus {
		fmt.Fprintf(app.Err, "feedback %s is already %s\n", id, preview.Draft.Status)
		return 1
	}
	if err := confirmAndSend(ctx, app, service, answers, preview); err != nil {
		fmt.Fprintln(app.Err, err)
		return 1
	}
	return 0
}

// sendPending asks about every pending draft in turn: each is shown and
// sent only on its own "y". A draft that fails doesn't stop the others;
// the exit code is then 1.
func sendPending(ctx context.Context, app *common.App, service handler.FeedbackService, answers *bufio.Reader) int {
	code := 0
	pending, err := service.Pending(ctx)
	if err != nil {
		fmt.Fprintln(app.Err, err)
		code = 1
	}
	if len(pending) == 0 && code == 0 {
		fmt.Fprintln(app.Out, "No feedback drafts to send.")
	}
	for i, preview := range pending {
		if i > 0 {
			fmt.Fprintln(app.Out)
		}
		fmt.Fprintf(app.Out, "Feedback %s (%d of %d)\n\n", preview.Draft.ID, i+1, len(pending))
		if err := confirmAndSend(ctx, app, service, answers, preview); err != nil {
			fmt.Fprintln(app.Err, err)
			code = 1
		}
	}
	return code
}

// confirmAndSend prints preview's issue and sends it if the next answer is
// "y" or "yes"; it prints what it did.
func confirmAndSend(ctx context.Context, app *common.App, service handler.FeedbackService, answers *bufio.Reader, preview handler.FeedbackPreview) error {
	printIssue(app.Out, preview)
	fmt.Fprint(app.Out, "\nOpen this issue? It is public if the repository is. [y/N] ")
	answer, err := answers.ReadString('\n')
	if err != nil && err != io.EOF {
		return err
	}
	if reply := strings.ToLower(strings.TrimSpace(answer)); reply != "y" && reply != "yes" {
		fmt.Fprintln(app.Out, "Not sent.")
		return nil
	}
	draft, err := service.Send(ctx, preview.Draft.ID, preview.Hash)
	if err != nil {
		return err
	}
	fmt.Fprintf(app.Out, "Sent: %s\n", draft.IssueURL)
	return nil
}

func readBody(app *common.App, file string) (string, error) {
	if file == "-" {
		raw, err := io.ReadAll(app.In)
		return string(raw), err
	}
	raw, err := app.ReadFile(file)
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

func (f *Feedback) sendCommand(id string) string {
	cmd := "ai-skill-manager feedback send " + id
	if f.Source.Config != "" {
		cmd += " -c " + f.Source.Config
	}
	return cmd
}
