// Command ai-skill-manager is the single composition root for both CLI names.
package main

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/command"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/internal/logging"
	"github.com/InsonusK/go-ai-skill-manage/internal/profiling"
	"github.com/InsonusK/go-ai-skill-manage/internal/version"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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
		stop, err := profiling.Start(opts.ProfileOutput)
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
	codec := document.Codec{}
	store := filesystem.Store{}
	detector := discovery.Detector{Codec: codec}
	sources := repository.Provider{Local: repository.Local{}, Remote: repository.Fetcher{
		Git:     repository.GitCloner{Runner: repository.GitProcess{}},
		Archive: repository.Archive{Client: &http.Client{Timeout: 60 * time.Second}},
	}}
	service := &services.SyncService{
		Sources: sources, Detector: detector, Relations: relations.Expander{Detector: detector},
		Planner: planning.Planner{State: store, Codec: codec}, Writer: store,
	}
	app := command.App{Sync: service, ReadFile: os.ReadFile, Out: os.Stdout, Err: os.Stderr, Version: version.Version}
	logger.Debug("command started")
	code = app.Execute(ctx, opts, cwd)
	logger.Debug("command finished", "exit_code", code)
	return code
}
