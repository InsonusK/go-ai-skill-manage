package config

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"go.yaml.in/yaml/v3"
)

// Config is a parsed configuration: the request, and what in the file the
// user should fix though it doesn't stop the work (deprecated settings).
type Config struct {
	Request  model.Request
	Warnings issues.ConfigIssues
}

// deprecations collects the "deprecated-setting" warnings of one Parse.
type deprecations struct{ list issues.ConfigIssues }

func (d *deprecations) add(setting, message string) {
	d.list = append(d.list, issues.ConfigIssue{Code: issues.CodeDeprecatedSetting, Setting: setting, Message: message})
}

type Overrides struct {
	Target                      string
	DryRun                      bool
	RemoveOrphans, AddRelations *bool
}

func Parse(data []byte) (Config, error) {
	req := model.Request{RemoveOrphans: true}
	warnings := &deprecations{}
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
	if _, set := settings["on_conflict"]; set {
		warnings.add("settings.on_conflict", "deprecated setting, remove it: skills with the same name are always an error")
	}
	if req.ExcludeFromChecks, err = globalExcludeFromChecks(settings, warnings); err != nil {
		return Config{}, err
	}
	target, rootTarget := root["target"]
	targetNode := fieldNode(node.Content[0], "target")
	legacyTarget, settingsTarget := settings["target"]
	if rootTarget && settingsTarget {
		return Config{}, fmt.Errorf("target cannot be defined both at root and in settings")
	}
	targetSetting := "target"
	if settingsTarget {
		targetSetting = "settings.target"
		target = legacyTarget
		targetNode = fieldNode(fieldNode(node.Content[0], "settings"), "target")
	}
	if req.Targets, err = parseTargets(target, targetNode, targetSetting, warnings); err != nil {
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
	for i, raw := range list {
		m, e := mapping(raw, "source")
		if e != nil {
			return Config{}, e
		}
		s, e := parseSource(m, fmt.Sprintf("sources[%d]", i), warnings)
		if e != nil {
			return Config{}, e
		}
		req.Sources = append(req.Sources, s)
	}
	return Config{Request: req, Warnings: warnings.list}, nil
}
