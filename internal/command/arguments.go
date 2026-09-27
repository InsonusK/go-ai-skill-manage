package command

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/config"
)

// Commands are the commands the CLI runs.
var Commands = []string{"sync", "validate"}

type Options struct {
	// Command is "sync" or "validate".
	Command                                       string
	Config, SourceType, SourcePath, ProfileOutput string
	// MemProfileOutput is where --profile writes the heap profile.
	MemProfileOutput              string
	Subpaths                      []string
	Override                      config.Overrides
	Help, Version, Debug, Profile bool
	// Force is the deprecated --force: without a hash to skip unchanged
	// skills every managed folder is rewritten anyway.
	Force bool
}

func Parse(args []string) (Options, error) {
	opts := Options{ProfileOutput: "ai-skill-manager.prof", MemProfileOutput: "ai-skill-manager.mem.prof"}
	command := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
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
		case "-c", "--config", "-t", "--type", "-p", "--path", "--subpath", "--target", "--profile-output", "--mem-profile-output":
			needsValue = true
		case "-h", "--help", "--version", "--debug", "--profile", "--dry-run", "-f", "--force", "--remove-orphans", "--keep-orphans", "--add-relations":
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
		}
	}
	if !opts.Help && !opts.Version && command == "" {
		return opts, fmt.Errorf("a command is required: %s", strings.Join(Commands, ", "))
	}
	switch opts.SourceType {
	case "", "local", "github", "auto", "flat", "directory":
	default:
		return opts, fmt.Errorf("unknown source type %q", opts.SourceType)
	}
	return opts, nil
}

const Usage = `Usage: aism [--debug] [--profile] <command> [options]
       ai-skill-manager <command> [options]

Commands:
  sync       Load the skills of the configured sources, check them and write
             them into every target folder
  validate   Check the configuration and the skills without writing anything

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
      --profile            Write Go CPU and heap profiles, log memory totals
      --profile-output FILE      CPU profile path (default: ai-skill-manager.prof)
      --mem-profile-output FILE  Heap profile path (default: ai-skill-manager.mem.prof)
      --version            Print the build version
  -h, --help               Show this help
  -f, --force              Deprecated, has no effect: every managed skill
                           folder is rewritten on each sync
`
