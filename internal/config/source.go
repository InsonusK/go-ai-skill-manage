package config

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"strings"
)

// parseSource reads one source; setting is where it is in the
// configuration, e.g. "sources[1]".
func parseSource(m map[string]any, setting string, warnings *deprecations) (model.SourceSpec, error) {
	s := model.SourceSpec{}
	var err error
	if s.Type, err = stringValue(m, "type", "local"); err != nil {
		return s, err
	}
	switch s.Type {
	case "auto", "flat", "directory":
		s.Type = "local"
	case "local", "github":
	default:
		return s, fmt.Errorf("unknown source type %q", s.Type)
	}
	if s.Path, err = stringValue(m, "path", ""); err != nil {
		return s, err
	}
	if strings.TrimSpace(s.Path) == "" {
		return s, fmt.Errorf("source path is required")
	}
	if s.Tree, err = stringValue(m, "tree", "master"); err != nil {
		return s, err
	}
	if _, set := m["name"]; set {
		warnings.add(setting+".name", "removed setting, it has no effect: a skill keeps the name from its frontmatter")
	}
	def := []string{}
	if s.Type == "github" {
		def = []string{"skills"}
	}
	if s.Subpaths, err = listValue(m["subpath"], "subpath", def); err != nil {
		return s, err
	}
	if s.Tags, err = listValue(m["tags"], "tags", nil); err != nil {
		return s, err
	}
	if s.ExcludeFromChecks, err = sourceExcludeFromChecks(m, s.Path, setting, warnings); err != nil {
		return s, err
	}
	return s, nil
}
