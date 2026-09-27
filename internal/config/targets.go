package config

import (
	"fmt"
	"log/slog"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"go.yaml.in/yaml/v3"
)

// LinkAdapter is the deprecated adapter that turned on link rewriting;
// skills are now always laid out with their links rewritten, so it is
// accepted with a warning and dropped.
const LinkAdapter = "link-adapter"

// adapters reads an adapters list. link-adapter is accepted with a
// warning and left out: it no longer changes anything.
func adapters(v any) ([]string, error) {
	list, err := listValue(v, "adapters", nil)
	if err != nil {
		return nil, err
	}
	out := []string{}
	seen := map[string]bool{}
	for _, a := range list {
		if a == LinkAdapter {
			slog.Warn("deprecated adapter, remove it: links are always rewritten", "adapter", a)
			continue
		}
		if a != "claude-property-adapter" {
			return nil, fmt.Errorf("unknown adapter %q", a)
		}
		if !seen[a] {
			out = append(out, a)
			seen[a] = true
		}
	}
	return out, nil
}
func parseTargets(v any, node *yaml.Node) ([]model.Target, error) {
	if v == nil {
		v = ".agents/skills"
	}
	if s, ok := v.(string); ok {
		if s == "" {
			return nil, fmt.Errorf("target path cannot be empty")
		}
		return []model.Target{{Name: "default", Path: s, Adapters: []string{}}}, nil
	}
	m, err := mapping(v, "target")
	if err != nil {
		return nil, err
	}
	each, err := mapping(m["for_each"], "for_each")
	if err != nil {
		return nil, err
	}
	shared, err := adapters(each["adapters"])
	if err != nil {
		return nil, err
	}
	names := []string{}
	if node != nil {
		for i := 0; i < len(node.Content); i += 2 {
			if n := node.Content[i].Value; n != "for_each" {
				names = append(names, n)
			}
		}
	}
	if len(names) == 0 {
		return []model.Target{{Name: "default", Path: ".agents/skills", Adapters: shared}}, nil
	}
	out := []model.Target{}
	for _, name := range names {
		entry, err := mapping(m[name], "target."+name)
		if err != nil {
			return nil, err
		}
		def := ""
		switch name {
		case "default":
			def = ".agents/skills"
		case "claude":
			def = ".claude/skills"
		}
		p, err := stringValue(entry, "path", def)
		if err != nil {
			return nil, err
		}
		if p == "" {
			return nil, fmt.Errorf("target %q requires path", name)
		}
		own, err := adapters(entry["adapters"])
		if err != nil {
			return nil, err
		}
		merged := append([]string{}, shared...)
		for _, a := range own {
			found := false
			for _, b := range merged {
				if a == b {
					found = true
				}
			}
			if !found {
				merged = append(merged, a)
			}
		}
		out = append(out, model.Target{Name: name, Path: p, Adapters: merged})
	}
	return out, nil
}
