package command

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/config"
)

// Commands are the commands the CLI runs.
var Commands = []string{"sync", "validate", "feedback", "mcp"}

// FeedbackActions are the actions of the feedback command.
var FeedbackActions = []string{"draft", "show", "send", "decline"}

// MCPActions are the actions of the mcp command; without one it serves.
var MCPActions = []string{"install", "uninstall"}

// DefaultMCPServerName is the server's name in .mcp.json.
const DefaultMCPServerName = "ai-skills"

type Options struct {
	// Command is "sync" or "validate".
	Command                                              string
	Config, SourceType, SourcePath, ProfileOutput, Color string
	// MemProfileOutput is where --profile writes the heap profile.
	MemProfileOutput              string
	Subpaths                      []string
	Override                      config.Overrides
	Help, Version, Debug, Profile bool
	// Force is the deprecated --force: without a hash to skip unchanged
	// skills every managed folder is rewritten anyway.
	Force bool
	// Feedback is what the feedback command was given.
	Feedback FeedbackOptions
	// MCP is what the mcp command was given.
	MCP MCPOptions
}

// MCPOptions: Action is "" (serve) or one of MCPActions; Name is the
// server's key in .mcp.json; Replace lets install overwrite another entry
// under that name. Neither reaches the server.
type MCPOptions struct {
	Action, Name string
	Replace      bool
}

// FeedbackOptions: Action is one of FeedbackActions; draft takes Skill,
// Kind, Title and Body or BodyFile ("-" for stdin); the others take ID.
type FeedbackOptions struct {
	Action, ID, Skill, Kind, Title, Body, BodyFile string
}

func Parse(args []string) (Options, error) {
	opts := Options{ProfileOutput: "ai-skill-manager.prof", MemProfileOutput: "ai-skill-manager.mem.prof", Color: "auto", MCP: MCPOptions{Name: DefaultMCPServerName}}
	command := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			if command == "feedback" && opts.Feedback.Action == "" {
				if !slices.Contains(FeedbackActions, arg) {
					return opts, fmt.Errorf("unknown feedback action %q: use %s", arg, strings.Join(FeedbackActions, ", "))
				}
				opts.Feedback.Action = arg
				continue
			}
			if command == "mcp" && opts.MCP.Action == "" {
				if !slices.Contains(MCPActions, arg) {
					return opts, fmt.Errorf("unknown mcp action %q: use %s, or none to serve", arg, strings.Join(MCPActions, ", "))
				}
				opts.MCP.Action = arg
				continue
			}
			if command == "feedback" && opts.Feedback.Action != "draft" && opts.Feedback.ID == "" {
				opts.Feedback.ID = arg
				continue
			}
			if command != "" {
				return opts, fmt.Errorf("unexpected argument %q", arg)
			}
			if !slices.Contains(Commands, arg) {
				return opts, fmt.Errorf("unknown command %q", arg)
			}
			command = arg
			opts.Command = arg
			continue
		}
		key, value, hasValue := strings.Cut(arg, "=")
		needsValue := false
		switch key {
		case "-c", "--config", "-t", "--type", "-p", "--path", "--subpath", "--target", "--profile-output", "--mem-profile-output",
			"--color", "--skill", "--kind", "--title", "--body", "--body-file", "--name":
			needsValue = true
		case "-h", "--help", "--version", "--debug", "--profile", "--dry-run", "-f", "--force", "--remove-orphans", "--keep-orphans", "--add-relations",
			"--replace":
		default:
			return opts, fmt.Errorf("unknown flag %q", key)
		}
		if needsValue {
			if !hasValue {
				if i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
					return opts, fmt.Errorf("%s requires a value", key)
				}
				i++
				value = args[i]
			}
			if value == "" {
				return opts, fmt.Errorf("%s requires a value", key)
			}
			switch key {
			case "-c", "--config":
				opts.Config = value
			case "-t", "--type":
				opts.SourceType = value
			case "-p", "--path":
				opts.SourcePath = value
			case "--subpath":
				opts.Subpaths = append(opts.Subpaths, value)
			case "--target":
				opts.Override.Target = value
			case "--profile-output":
				opts.ProfileOutput = value
			case "--mem-profile-output":
				opts.MemProfileOutput = value
			case "--color":
				opts.Color = value
			case "--skill":
				opts.Feedback.Skill = value
			case "--kind":
				opts.Feedback.Kind = value
			case "--title":
				opts.Feedback.Title = value
			case "--body":
				opts.Feedback.Body = value
			case "--body-file":
				opts.Feedback.BodyFile = value
			case "--name":
				opts.MCP.Name = value
			}
			continue
		}
		enabled := true
		if hasValue {
			var err error
			enabled, err = strconv.ParseBool(value)
			if err != nil {
				return opts, fmt.Errorf("%s requires a boolean", key)
			}
		}
		switch key {
		case "-h", "--help":
			opts.Help = enabled
		case "--version":
			opts.Version = enabled
		case "--debug":
			opts.Debug = enabled
		case "--profile":
			opts.Profile = enabled
		case "--dry-run":
			opts.Override.DryRun = enabled
		case "-f", "--force":
			opts.Force = enabled
		case "--remove-orphans":
			if enabled {
				v := true
				opts.Override.RemoveOrphans = &v
			}
		case "--keep-orphans":
			if enabled && (opts.Override.RemoveOrphans == nil || !*opts.Override.RemoveOrphans) {
				v := false
				opts.Override.RemoveOrphans = &v
			}
		case "--add-relations":
			opts.Override.AddRelations = &enabled
		case "--replace":
			opts.MCP.Replace = enabled
		}
	}
	if !opts.Help && !opts.Version && command == "" {
		return opts, fmt.Errorf("a command is required: %s", strings.Join(Commands, ", "))
	}
	if !slices.Contains([]string{"auto", "always", "never"}, opts.Color) {
		return opts, fmt.Errorf("--color must be auto, always or never")
	}
	if err := checkFeedback(opts); err != nil {
		return opts, err
	}
	if err := checkMCP(opts, args); err != nil {
		return opts, err
	}
	switch opts.SourceType {
	case "", "local", "github", "auto", "flat", "directory":
	default:
		return opts, fmt.Errorf("unknown source type %q", opts.SourceType)
	}
	return opts, nil
}

// checkFeedback: the feedback flags belong to feedback draft only, and
// every action has what it needs.
func checkFeedback(opts Options) error {
	f := opts.Feedback
	draftFlags := f.Skill != "" || f.Kind != "" || f.Title != "" || f.Body != "" || f.BodyFile != ""
	if opts.Help || opts.Version {
		return nil
	}
	if opts.Command != "feedback" || f.Action != "draft" {
		if draftFlags {
			return fmt.Errorf("--skill, --kind, --title, --body and --body-file are for feedback draft only")
		}
		if opts.Command != "feedback" {
			return nil
		}
	}
	switch {
	case f.Action == "":
		return fmt.Errorf("feedback needs an action: %s", strings.Join(FeedbackActions, ", "))
	case f.Action != "draft" && f.ID == "":
		return fmt.Errorf("feedback %s needs the draft id", f.Action)
	case f.Action == "draft" && (f.Skill == "" || f.Kind == "" || f.Title == ""):
		return fmt.Errorf("feedback draft needs --skill, --kind and --title")
	case f.Action == "draft" && (f.Body == "") == (f.BodyFile == ""):
		return fmt.Errorf("feedback draft needs either --body or --body-file")
	}
	return nil
}

// checkMCP: --name belongs to mcp install/uninstall, --replace to install;
// the server set up by install reads a config file, not --type/--path.
func checkMCP(opts Options, args []string) error {
	if opts.Help || opts.Version {
		return nil
	}
	named := slices.ContainsFunc(args, func(a string) bool { return a == "--name" || strings.HasPrefix(a, "--name=") })
	switch {
	case named && (opts.Command != "mcp" || opts.MCP.Action == ""):
		return fmt.Errorf("--name is for mcp install and mcp uninstall only")
	case opts.MCP.Replace && opts.MCP.Action != "install":
		return fmt.Errorf("--replace is for mcp install only")
	case opts.Command == "mcp" && opts.SourceType != "":
		return fmt.Errorf("mcp works with a config file (-c), not --type/--path")
	}
	return nil
}

const Usage = `Usage: aism [--debug] [--color MODE] [--profile] <command> [options]
       ai-skill-manager <command> [options]

Commands:
  sync       Load the skills of the configured sources, check them and write
             them into every target folder
  validate   Check the configuration and the skills without writing anything
  feedback   Report a bug or suggest an improvement to a skill's source
             (a GitHub issue), sent only after the user confirms it:
    feedback draft --skill NAME --kind bug|improvement --title TEXT
                   (--body TEXT | --body-file FILE|-)
               Write the draft to .ai-skills/feedback/ next to the config
    feedback show ID     Show the draft and the issue it would open
    feedback send ID     Show it, ask for confirmation, open the issue
                         (asks in a terminal only: the user runs it)
    feedback decline ID  Close the draft without sending; the file stays
  mcp        Serve the feedback tools to an agent over MCP (stdio): the
             agent drafts, the user confirms in the client's dialog
    mcp install [--name ai-skills] [--replace]
               Add this server to .mcp.json next to the config (Claude Code);
               -c, when given, is passed to the server too
    mcp uninstall [--name ai-skills]
               Remove it from .mcp.json

Options:
  -c, --config FILE        YAML or JSON config (default: ai-skills.yaml)
  -t, --type TYPE          Source without a config: local or github
  -p, --path PATH          Source path or "repository-url branch"
      --subpath PATH       Repository subpath; repeatable
      --target PATH        Write into this folder instead of the configured targets
      --dry-run            Plan the changes and print them without writing
      --remove-orphans     Remove managed skill folders no longer synchronized
      --keep-orphans       Keep them
      --add-relations      Also load the skills that selected skills link to
      --debug              Structured debug logs on stderr
      --color MODE         Color output: auto, always or never (default: auto)
      --profile            Write Go CPU and heap profiles, log memory totals
      --profile-output FILE      CPU profile path (default: ai-skill-manager.prof)
      --mem-profile-output FILE  Heap profile path (default: ai-skill-manager.mem.prof)
      --version            Print the build version
  -h, --help               Show this help
  -f, --force              Deprecated, has no effect: every managed skill
                           folder is rewritten on each sync
`
