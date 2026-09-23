package transform

import (
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

var native = map[string]bool{"name": true, "description": true, "when_to_use": true, "argument-hint": true, "arguments": true, "disable-model-invocation": true, "user-invocable": true, "allowed-tools": true, "disallowed-tools": true, "model": true, "effort": true, "context": true, "agent": true, "hooks": true, "paths": true, "shell": true}

func Claude(doc model.SkillDocument) model.SkillDocument {
	if !doc.HasFrontmatter {
		return doc
	}
	props := map[string]any{}
	extras := map[string]any{}
	for k, v := range doc.Properties {
		if native[k] {
			props[k] = v
		} else {
			extras[k] = v
		}
	}
	if value, ok := extras["whenToUse"]; ok && value != nil {
		if list, ok := value.([]any); ok {
			parts := []string{}
			for _, v := range list {
				parts = append(parts, fmt.Sprint(v))
			}
			value = strings.Join(parts, ",")
		}
		if _, present := props["when_to_use"]; !present {
			props["when_to_use"] = value
			delete(extras, "whenToUse")
		} else {
			extras["whenToUse"] = value
		}
	}
	doc.Properties = props
	doc.Metadata = extras
	return doc
}
