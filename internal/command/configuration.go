package command

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/config"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"path/filepath"
	"strings"
)

// Request resolves opts (plus, when applicable, the ai-skills.yaml config
// file it points at) into a model.Request. Exported so main.go can resolve
// it once, up front -- before constructing a sourcing.Manager, which needs
// the resolved TempDir -- instead of only inside Execute.
func (a App) Request(opts Options, cwd string) (model.Request, error) {
	base := cwd
	var cfg config.Config
	if opts.Config != "" || opts.SourceType == "" {
		filename := opts.Config
		if filename == "" {
			filename = "ai-skills.yaml"
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
		if strings.TrimSpace(opts.SourcePath) == "" {
			return model.Request{}, fmt.Errorf("--path is required when using --type")
		}
		var err error
		cfg, err = config.Parse([]byte("sources: []"))
		if err != nil {
			return model.Request{}, err
		}
		source := model.SourceSpec{Type: opts.SourceType, Path: opts.SourcePath, Tree: "master"}
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
			source.Subpaths = opts.Subpaths
			if len(source.Subpaths) == 0 {
				source.Subpaths = []string{"skills"}
			}
		}
		cfg.Request.Sources = []model.SourceSpec{source}
	}
	return config.Resolve(cfg, opts.Override, base)
}
