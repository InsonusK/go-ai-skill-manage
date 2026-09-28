// Command ai-skill-manager is the single composition root for both CLI names.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/command"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/tracker"
	"github.com/InsonusK/go-ai-skill-manage/internal/logging"
	"github.com/InsonusK/go-ai-skill-manage/internal/profiling"
	"github.com/InsonusK/go-ai-skill-manage/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() { os.Exit(run()) }
func run() (code int) {
	opts, err := command.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	logger := logging.Init(os.Stderr, opts.Debug)
	if opts.Profile && !opts.Help && !opts.Version {
		stop, err := profiling.Start(opts.ProfileOutput, opts.MemProfileOutput)
		if err != nil {
			logger.Error("profiling", "error", err)
			return 1
		}
		defer func() {
			if err := stop(); err != nil {
				logger.Error("close profile", "error", err)
				code = 1
			}
		}()
	}
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("working directory", "error", err)
		return 1
	}
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); opts.Command == "mcp" && dir != "" {
		// Claude Code gives the MCP server the project's root: the config
		// (and so the drafts) are found from there, whatever the server's
		// working directory.
		cwd = dir
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	entity.SetDefaultLinkSearcher(links.NewDefaultLinkFactory())
	store := filesystem.Store{}
	providers := map[string]interfaces.SourceProvider{
		"local": repository.Local{},
		"github": repository.Fetcher{
			Git:     repository.GitCloner{Runner: repository.GitProcess{}},
			Archive: repository.Archive{Client: &http.Client{Timeout: 60 * time.Second}},
		},
	}
	github := tracker.GitHub{
		Client: &http.Client{Timeout: 30 * time.Second},
		Token:  tracker.TokenSource{Getenv: os.Getenv, GH: tracker.GHAuthToken}.Token,
	}
	app := command.App{
		Providers: providers, State: store, Writer: store, ReadFile: os.ReadFile, Out: os.Stdout, Err: os.Stderr, Version: version.Version,
		Markers:      store,
		Drafts:       func(dir string) interfaces.FeedbackDrafts { return filesystem.FeedbackDrafts{Dir: dir} },
		Trackers:     map[string]interfaces.IssueTracker{"github": github},
		In:           os.Stdin,
		IsTerminal:   stdinIsTerminal,
		MCPTransport: &mcp.StdioTransport{},
	}
	logger.Debug("command started")
	code = app.Execute(ctx, opts, cwd)
	logger.Debug("command finished", "exit_code", code)
	return code
}

// stdinIsTerminal tells whether stdin is a character device (a terminal),
// not a pipe or a file, as an agent's shell gives. /dev/null passes too,
// but then the answer is empty: not sent.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
