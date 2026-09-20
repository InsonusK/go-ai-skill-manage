package config

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"strings"
)

func parseSource(m map[string]any) (model.SourceSpec, error) {
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
	if s.Name, err = stringValue(m, "name", ""); err != nil {
		return s, err
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
	if s.SkipFolders, err = listValue(m["skip_folder"], "skip_folder", []string{"examples"}); err != nil {
		return s, err
	}
	return s, nil
}
