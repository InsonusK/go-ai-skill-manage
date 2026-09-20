package command

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"strconv"
	"strings"
)

type Options struct {
	Config, SourceType, SourcePath, ProfileOutput string
	Subpaths                                      []string
	Override                                      config.Overrides
	Help, Version, Debug, Profile                 bool
}

func Parse(args []string) (Options, error) {
	opts := Options{ProfileOutput: "ai-skill-manager.prof"}
	command := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			if command != "" {
				return opts, fmt.Errorf("unexpected argument %q", arg)
			}
			if arg != "sync" {
				return opts, fmt.Errorf("unknown command %q", arg)
			}
			command = arg
			continue
		}
		key, value, hasValue := strings.Cut(arg, "=")
		needsValue := false
		switch key {
		case "-c", "--config", "-t", "--type", "-p", "--path", "--subpath", "--target", "--profile-output":
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
			opts.Override.Force = enabled
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
		return opts, fmt.Errorf("a command is required: sync")
	}
	switch opts.SourceType {
	case "", "local", "github", "auto", "flat", "directory":
	default:
		return opts, fmt.Errorf("unknown source type %q", opts.SourceType)
	}
	return opts, nil
}

const Usage = `Usage: ai-skill-manager [--debug] [--profile] sync [options]
       aism sync [options]

Synchronize AI skills from local directories or Git repositories.

  -c, --config FILE        YAML or JSON config (default: ai-skills.yaml)
  -t, --type TYPE          local or github (legacy: auto, flat, directory)
  -p, --path PATH          Source path or "repository-url branch"
      --subpath PATH       Repository subpath; repeatable
      --target PATH        Override all configured targets
      --dry-run            Validate and show changes without writing targets
  -f, --force              Recopy unchanged skills
      --remove-orphans     Remove previously managed skills no longer selected
      --keep-orphans       Keep obsolete skills
      --add-relations      Include skills referenced outside selected paths
      --debug              Enable structured debug logs on stderr
      --profile            Write a Go CPU profile
      --profile-output FILE  Profile path (default: ai-skill-manager.prof)
      --version            Print build version
  -h, --help               Show this help
`
