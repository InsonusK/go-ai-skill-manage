// Command ai-skill-manager is the single composition root for both CLI names.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/InsonusK/go-ai-skill-manager/internal/command"
	"github.com/InsonusK/go-ai-skill-manager/internal/command/common"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manager/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manager/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manager/internal/infrastructure/tracker"
	"github.com/InsonusK/go-ai-skill-manager/internal/logging"
	"github.com/InsonusK/go-ai-skill-manager/internal/profiling"
	"github.com/InsonusK/go-ai-skill-manager/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() { os.Exit(run()) }
func run() (code int) {
	inv, err := command.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	global := inv.Global
	color := logging.ColorEnabled(global.Color, fileIsTerminal(os.Stderr), os.Getenv("NO_COLOR"))
	logger := logging.Init(os.Stderr, global.Debug, color)
	if global.Profile && !global.Help && !global.Version {
		stop, err := profiling.Start(global.ProfileOutput, global.MemProfileOutput)
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
	app := &common.App{
		Providers: providers, State: store, Writer: store, ReadFile: os.ReadFile, Out: os.Stdout, Err: os.Stderr, Version: version.Version,
		Color:        color,
		Getenv:       os.Getenv,
		Markers:      store,
		Drafts:       func(dir string) interfaces.FeedbackDrafts { return filesystem.FeedbackDrafts{Dir: dir} },
		Trackers:     map[string]interfaces.IssueTracker{"github": github},
		In:           os.Stdin,
		IsTerminal:   stdinIsTerminal,
		MCPTransport: &mcp.StdioTransport{},
		WriteFile:    os.WriteFile,
		Self:         self,
	}
	logger.Debug("command started")
	code = command.Execute(ctx, app, inv, cwd)
	logger.Debug("command finished", "exit_code", code)
	return code
}

// stdinIsTerminal tells whether stdin is a character device (a terminal),
// not a pipe or a file, as an agent's shell gives. /dev/null passes too,
// but then the answer is empty: not sent.
func stdinIsTerminal() bool {
	return fileIsTerminal(os.Stdin)
}

func fileIsTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// self is how .mcp.json should run this tool: the name it was run under
// (aism or ai-skill-manager) when that name finds this same file in PATH,
// else this file's absolute path.
func self() (string, bool) {
	path, err := os.Executable()
	if err != nil {
		return "ai-skill-manager", false
	}
	name := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	if found, err := exec.LookPath(name); err == nil {
		a, errA := os.Stat(found)
		b, errB := os.Stat(path)
		if errA == nil && errB == nil && os.SameFile(a, b) {
			return name, true
		}
	}
	return path, false
}
