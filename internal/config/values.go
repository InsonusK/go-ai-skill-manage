package config

import (
	"fmt"
	"go.yaml.in/yaml/v3"
	"strings"
)

func literalTags(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode && strings.HasPrefix(n.Tag, "!") && !strings.HasPrefix(n.Tag, "!!") {
		n.Value = n.Tag + n.Value
		n.Tag = "!!str"
	}
	for _, c := range n.Content {
		literalTags(c)
	}
}
func fieldNode(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}
func mapping(v any, name string) (map[string]any, error) {
	if v == nil {
		return map[string]any{}, nil
	}
	if m, ok := v.(map[string]any); ok {
		return m, nil
	}
	return nil, fmt.Errorf("%s must be a mapping", name)
}
func stringValue(m map[string]any, k, def string) (string, error) {
	v, ok := m[k]
	if !ok || v == nil {
		return def, nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", k)
	}
	return s, nil
}
func boolean(m map[string]any, k string, def bool) (bool, error) {
	v, ok := m[k]
	if !ok {
		return def, nil
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", k)
	}
	return b, nil
}
func listValue(v any, name string, def []string) ([]string, error) {
	if v == nil {
		return append([]string{}, def...), nil
	}
	if s, ok := v.(string); ok {
		return []string{s}, nil
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a string or list", name)
	}
	out := []string{}
	for _, value := range list {
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s entries must be strings", name)
		}
		out = append(out, s)
	}
	return out, nil
}
