package config

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"go.yaml.in/yaml/v3"
)

type Config struct{ Request model.Request }
type Overrides struct {
	Target                      string
	DryRun, Force               bool
	RemoveOrphans, AddRelations *bool
}

func Parse(data []byte) (Config, error) {
	req := model.Request{RemoveOrphans: true, Conflict: "error"}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if len(node.Content) == 0 {
		return Config{}, fmt.Errorf("config must be a mapping")
	}
	literalTags(&node)
	var root map[string]any
	if err := node.Decode(&root); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if root == nil {
		return Config{}, fmt.Errorf("config must be a mapping")
	}
	settings, err := mapping(root["settings"], "settings")
	if err != nil {
		return Config{}, err
	}
	if req.TempDir, err = stringValue(settings, "temp_dir", ""); err != nil {
		return Config{}, err
	}
	if req.DryRun, err = boolean(settings, "dry_run", false); err != nil {
		return Config{}, err
	}
	if req.RemoveOrphans, err = boolean(settings, "remove_orphans", true); err != nil {
		return Config{}, err
	}
	if req.AddRelations, err = boolean(settings, "add_relations", false); err != nil {
		return Config{}, err
	}
	if req.Conflict, err = stringValue(settings, "on_conflict", "error"); err != nil {
		return Config{}, err
	}
	if req.Conflict != "error" && req.Conflict != "last_wins" {
		return Config{}, fmt.Errorf("unknown on_conflict %q", req.Conflict)
	}
	if req.ExcludeFromChecks, err = globalExcludeFromChecks(settings); err != nil {
		return Config{}, err
	}
	target, rootTarget := root["target"]
	targetNode := fieldNode(node.Content[0], "target")
	legacyTarget, settingsTarget := settings["target"]
	if rootTarget && settingsTarget {
		return Config{}, fmt.Errorf("target cannot be defined both at root and in settings")
	}
	if settingsTarget {
		target = legacyTarget
		targetNode = fieldNode(fieldNode(node.Content[0], "settings"), "target")
	}
	if req.Targets, err = parseTargets(target, targetNode); err != nil {
		return Config{}, err
	}
	sources := root["sources"]
	if sources == nil {
		sources = []any{}
	}
	list, ok := sources.([]any)
	if !ok {
		return Config{}, fmt.Errorf("sources must be a list")
	}
	req.Sources = []model.SourceSpec{}
	for _, raw := range list {
		m, e := mapping(raw, "source")
		if e != nil {
			return Config{}, e
		}
		s, e := parseSource(m)
		if e != nil {
			return Config{}, e
		}
		req.Sources = append(req.Sources, s)
	}
	return Config{Request: req}, nil
}
