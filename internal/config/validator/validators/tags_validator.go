package validators

import (
	"context"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/tags"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// TagsValidator checks that every tag expression of every source parses.
type TagsValidator struct{}

var _ ConfigValidator = TagsValidator{}

func (TagsValidator) Name() string { return "tags-validator" }

func (TagsValidator) DependsOn() []validator.Dependency { return nil }

// Validate reports "invalid-tags" for each expression that doesn't parse.
//
// Пример: sources[1].tags = ["go", "(cli"] -> одна проблема с Setting
// "sources[1].tags[1]".
func (TagsValidator) Validate(ctx context.Context, req model.Request) []issues.ConfigIssue {
	var problems []issues.ConfigIssue
	for i, spec := range req.Sources {
		for j, expr := range spec.Tags {
			if _, err := tags.Match(nil, []string{expr}); err != nil {
				problems = append(problems, issues.ConfigIssue{Code: "invalid-tags", Source: spec.Key().String(), Setting: fmt.Sprintf("sources[%d].tags[%d]", i, j), Message: fmt.Sprintf("%v: %q", err, expr)})
			}
		}
	}
	return problems
}
