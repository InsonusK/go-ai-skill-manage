package common

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	configvalidator "github.com/InsonusK/go-ai-skill-manage/internal/config/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// DefaultConfigFile is the config read without --config.
const DefaultConfigFile = "ai-skills.yaml"

// Source is where a command takes its request from: the config file, or
// (direct mode, --type and --path) one source without a config.
type Source struct {
	Config, Type, Path string
	Subpaths           []string
}

// ConfigFlag is -c: what feedback and mcp take.
func (s *Source) ConfigFlag() Flag {
	return String(&s.Config, "FILE", "YAML or JSON config (default: "+DefaultConfigFile+");\nits folder is the project's root", "-c", "--config")
}

// Flags are -c and the direct mode's flags.
func (s *Source) Flags() []Flag {
	return []Flag{
		s.ConfigFlag(),
		Func("TYPE", "Source without a config: local or github", func(v string) error {
			switch v {
			case "local", "github", "auto", "flat", "directory":
				s.Type = v
				return nil
			}
			return fmt.Errorf("unknown source type %q", v)
		}, "-t", "--type"),
		String(&s.Path, "PATH", `Source path, for github "repository-url [branch|tag]"`, "-p", "--path"),
		Strings(&s.Subpaths, "PATH", "Repository subpath of a github source (default: skills);\nrepeatable", "--subpath"),
	}
}

// AddRelationsFlag is --add-relations.
func AddRelationsFlag(o *config.Overrides) Flag {
	return BoolFunc("Also load the skills that selected skills link to", func(v bool) { o.AddRelations = &v }, "--add-relations")
}

// Request resolves src (plus, when applicable, the config file it points
// at) and override into a model.Request.
func (a *App) Request(src Source, override config.Overrides, cwd string) (model.Request, error) {
	base := cwd
	var cfg config.Config
	if src.Config != "" || src.Type == "" {
		filename := src.Config
		if filename == "" {
			filename = DefaultConfigFile
		}
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(cwd, filename)
		}
		data, err := a.ReadFile(filename)
		if err != nil {
			return model.Request{}, err
		}
		cfg, err = config.Parse(data)
		if err != nil {
			return model.Request{}, err
		}
		base = filepath.Dir(filename)
	} else {
		if strings.TrimSpace(src.Path) == "" {
			return model.Request{}, fmt.Errorf("--path is required when using --type")
		}
		var err error
		cfg, err = config.Parse([]byte("sources: []"))
		if err != nil {
			return model.Request{}, err
		}
		source := model.SourceSpec{Type: src.Type, Path: src.Path, Tree: "master"}
		if source.Type == "auto" || source.Type == "flat" || source.Type == "directory" {
			fmt.Fprintf(a.Err, "Source type %s is deprecated; use local\n", source.Type)
			source.Type = "local"
		}
		if source.Type == "github" {
			parts := strings.Fields(source.Path)
			source.Path = parts[0]
			if len(parts) > 1 {
				source.Tree = strings.Join(parts[1:], " ")
			}
			source.Subpaths = src.Subpaths
			if len(source.Subpaths) == 0 {
				source.Subpaths = []string{"skills"}
			}
		}
		cfg.Request.Sources = []model.SourceSpec{source}
	}
	return config.Resolve(cfg, override, base)
}

// LoadRequest is Request, then the configuration's check (config/validator
// -- the domain trusts the request after that). On failure it prints the
// error or the problems (as a tree) and returns false.
func (a *App) LoadRequest(ctx context.Context, src Source, override config.Overrides, cwd string) (model.Request, bool) {
	req, err := a.Request(src, override, cwd)
	if err != nil {
		fmt.Fprintln(a.Err, err)
		return model.Request{}, false
	}
	if problems := configvalidator.Validate(ctx, req); len(problems) > 0 {
		PrintIssues(a.Err, Reportables(problems), a.Color)
		return model.Request{}, false
	}
	return req, true
}
