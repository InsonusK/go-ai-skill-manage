// Package command is the command line: command.go picks the command and
// takes the global options, each command's file parses, runs and explains
// that command, common holds what they share.
package command

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/command/common"
)

// commands are the commands of the CLI, new and empty for each parse, in
// the order of the help.
func commands() []common.Command {
	return []common.Command{&Sync{}, &Validate{}, &Feedback{}, &MCP{}}
}

// Global are the options every command takes, before or after its name.
type Global struct {
	Help, Version, Debug, Profile          bool
	Color, ProfileOutput, MemProfileOutput string
}

func newGlobal() Global {
	return Global{Color: "auto", ProfileOutput: "ai-skill-manager.prof", MemProfileOutput: "ai-skill-manager.mem.prof"}
}

func (g *Global) flags() []common.Flag {
	return []common.Flag{
		common.Bool(&g.Debug, "Structured debug logs on stderr", "--debug"),
		common.Func("MODE", "Color output: auto, always or never (default: auto);\nauto colors a terminal only and honors NO_COLOR", func(v string) error {
			if !slices.Contains([]string{"auto", "always", "never"}, v) {
				return fmt.Errorf("must be auto, always or never")
			}
			g.Color = v
			return nil
		}, "--color"),
		common.Bool(&g.Profile, "Write Go CPU and heap profiles, log memory totals", "--profile"),
		common.String(&g.ProfileOutput, "FILE", "CPU profile path (default: ai-skill-manager.prof)", "--profile-output"),
		common.String(&g.MemProfileOutput, "FILE", "Heap profile path (default: ai-skill-manager.mem.prof)", "--mem-profile-output"),
		common.Bool(&g.Version, "Print the build version", "--version"),
		common.Bool(&g.Help, "Show the help (of the command, when one is given)", "-h", "--help"),
	}
}

// Invocation is a parsed command line.
type Invocation struct {
	Global Global
	// Command is nil when none is named: --help or --version alone.
	Command common.Command
}

// Parse finds the command (the first argument that isn't a global flag or
// its value) and lets it parse the other arguments together with the
// global flags. Before the command only global flags may stand. With
// --help, the command's own errors don't matter: its help is printed.
//
// Пример: ["--debug", "sync", "-c", "x.yaml", "--color", "never"] ->
// Global{Debug, Color: never}, Sync{Source.Config: x.yaml}.
func Parse(args []string) (Invocation, error) {
	inv := Invocation{Global: newGlobal()}
	global := inv.Global.flags()
	at, err := commandAt(args, global)
	if err != nil {
		return inv, err
	}
	if at < 0 {
		if _, err := common.ParseFlags(args, global, "aism"); err != nil {
			return inv, err
		}
		if !inv.Global.Help && !inv.Global.Version {
			return inv, fmt.Errorf("a command is required: %s", strings.Join(commandNames(), ", "))
		}
		return inv, nil
	}
	i := slices.IndexFunc(commands(), func(c common.Command) bool { return c.Name() == args[at] })
	if i < 0 {
		return inv, fmt.Errorf("unknown command %q: use %s", args[at], strings.Join(commandNames(), ", "))
	}
	inv.Command = commands()[i]
	rest := slices.Concat(args[:at], args[at+1:])
	if err := inv.Command.Parse(rest, global); err != nil && !inv.Global.Help {
		return inv, err
	}
	return inv, nil
}

// commandAt is the index of the command's name in args, -1 if none.
func commandAt(args []string, global []common.Flag) (int, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			return i, nil
		}
		name, _, hasValue := strings.Cut(arg, "=")
		flag := common.Lookup(global, name)
		if flag == nil {
			return -1, fmt.Errorf("unknown flag %q: a command's flags go after the command; see aism --help", name)
		}
		if flag.TakesValue() && !hasValue {
			i++
		}
	}
	return -1, nil
}

func commandNames() []string {
	var names []string
	for _, c := range commands() {
		names = append(names, c.Name())
	}
	return names
}

// Help is the help of aism itself: its commands and the global options.
func Help() string {
	g := newGlobal()
	h := common.Help{
		Usage:        []string{"aism [global options] <command> [options]", "ai-skill-manager [global options] <command> [options]"},
		Description:  "Synchronize AI agent skills from their sources (local folders, GitHub)\ninto the project's skill folders.",
		EntriesTitle: "Commands",
		Global:       g.flags(),
		Footer:       "Run \"aism <command> --help\" for a command's options.",
	}
	for _, c := range commands() {
		h.Entries = append(h.Entries, [2]string{c.Name(), c.Summary()})
	}
	return h.String()
}

// Execute runs inv and returns the exit code: 0 on success, 1 when the
// configuration, the skills or a target have problems (printed on Err) or
// the command fails.
func Execute(ctx context.Context, app *common.App, inv Invocation, cwd string) int {
	switch {
	case inv.Global.Help && inv.Command == nil:
		fmt.Fprint(app.Out, Help())
		return 0
	case inv.Global.Help:
		g := newGlobal()
		fmt.Fprint(app.Out, inv.Command.Help(g.flags()))
		return 0
	case inv.Global.Version:
		fmt.Fprintln(app.Out, app.Version)
		return 0
	}
	return inv.Command.Run(ctx, app, cwd)
}
