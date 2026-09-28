package command

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	configvalidator "github.com/InsonusK/go-ai-skill-manage/internal/config/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// App runs one command line: it resolves the request from the options and
// the config file, checks the configuration (config/validator -- the
// domain trusts the request after that), then runs the command. It makes
// the source manager itself, once the request's temporary folder is known.
type App struct {
	// Providers acquire repositories by source type ("local", "github").
	Providers map[string]interfaces.SourceProvider
	State     interfaces.StateReader
	Writer    interfaces.PlanWriter
	ReadFile  func(string) ([]byte, error)
	Out, Err  io.Writer
	Version   string

	// The feedback command's ports: Markers read skills' markers in the
	// targets, Drafts keeps drafts in a folder, Trackers open issues by
	// source type.
	Markers  interfaces.MarkerReader
	Drafts   func(dir string) interfaces.FeedbackDrafts
	Trackers map[string]interfaces.IssueTracker
	// In is the user's input: the body of feedback draft --body-file -,
	// the answer to feedback send.
	In io.Reader
	// IsTerminal tells whether In is the user's terminal; feedback send
	// asks only there.
	IsTerminal func() bool
	// Now is the clock; nil means time.Now.
	Now func() time.Time
	// MCPTransport is what the mcp command serves on (stdio).
	MCPTransport mcp.Transport
}

// Execute runs opts and returns the exit code: 0 on success, 1 when the
// configuration, the skills or a target have problems (printed as a tree
// on Err) or the command fails.
func (a App) Execute(ctx context.Context, opts Options, cwd string) (code int) {
	if opts.Help {
		fmt.Fprint(a.Out, Usage)
		return 0
	}
	if opts.Version {
		fmt.Fprintln(a.Out, a.Version)
		return 0
	}
	if opts.Force {
		slog.WarnContext(ctx, "deprecated flag, remove it: every managed skill folder is rewritten on each sync", "flag", "--force")
	}
	if opts.Command == "mcp" {
		return a.serveMCP(ctx, opts, cwd)
	}
	req, err := a.Request(opts, cwd)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return 1
	}
	if problems := configvalidator.Validate(ctx, req); len(problems) > 0 {
		PrintIssues(a.Err, reportables(problems))
		return 1
	}
	if opts.Command == "feedback" {
		return a.feedback(ctx, opts, req)
	}
	sources := sourcing.NewManager(a.Providers, req.TempDir)
	defer func() {
		if err := sources.Close(ctx); err != nil {
			fmt.Fprintln(a.Err, "close sources:", err)
			code = 1
		}
	}()
	switch opts.Command {
	case "validate":
		catalog, problems := handler.FetchAndValidateSkills(ctx, sources, req)
		if len(problems) > 0 {
			PrintIssues(a.Err, reportables(problems))
			return 1
		}
		fmt.Fprintf(a.Out, "Checked %d skill(s): no problems\n", len(catalog.Skills()))
		return 0
	default:
		result, err := handler.SyncService{Sources: sources, State: a.State, Writer: a.Writer}.Run(ctx, req)
		if err != nil {
			if rows := reportables(err); rows != nil {
				PrintIssues(a.Err, rows)
			} else {
				fmt.Fprintln(a.Err, err)
			}
			return 1
		}
		PrintResult(a.Out, result)
		return 0
	}
}
