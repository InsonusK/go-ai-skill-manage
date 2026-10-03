// Package common holds what the commands of the CLI share: the flag
// table, the ports they run with (App), loading the request from the
// config, printing problems.
package common

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Command is one command of the CLI (sync, feedback, ...): it parses its
// arguments, prints its help and runs.
type Command interface {
	Name() string
	// Summary is one line for the list of commands.
	Summary() string
	// Parse reads the arguments after the command's name; global are the
	// flags every command takes too.
	Parse(args []string, global []Flag) error
	// Help is the command's help; after Parse, of the action it was given.
	Help(global []Flag) string
	Run(ctx context.Context, app *App, cwd string) int
}

// App is what the commands run with: the ports to the outside world.
type App struct {
	// Providers acquire repositories by source type ("local", "github").
	Providers map[string]interfaces.SourceProvider
	State     interfaces.StateReader
	Writer    interfaces.PlanWriter
	ReadFile  func(string) ([]byte, error)
	Out, Err  io.Writer
	Version   string
	// Color enables ANSI styling for human-facing output.
	Color bool
	// Getenv reads the environment; nil reads nothing.
	Getenv func(string) string

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
	// WriteFile writes .mcp.json for mcp install/uninstall.
	WriteFile func(string, []byte, os.FileMode) error
	// Self is how .mcp.json should run this tool: the name it runs under,
	// when that name finds it in PATH (inPath), else its absolute path.
	Self func() (command string, inPath bool)
}

// Env is the environment variable name, "" without Getenv.
func (a *App) Env(name string) string {
	if a.Getenv == nil {
		return ""
	}
	return a.Getenv(name)
}
