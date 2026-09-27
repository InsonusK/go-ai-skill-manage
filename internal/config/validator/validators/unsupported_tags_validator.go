package validators

import (
	"context"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// UnsupportedTagsValidator rejects tags: selecting skills by tags is
// postponed (see AGENTS.md), and until it is back the selector ignores
// them -- a source with tags would silently select every skill under its
// subpaths. Remove this validator when the tag filter is back; the tag
// expressions stay checked by TagsValidator.
type UnsupportedTagsValidator struct{}

var _ ConfigValidator = UnsupportedTagsValidator{}

func (UnsupportedTagsValidator) Name() string { return "unsupported-tags-validator" }

func (UnsupportedTagsValidator) DependsOn() []validator.Dependency { return nil }

// Validate reports "unsupported-tags" for each source with tags.
func (UnsupportedTagsValidator) Validate(ctx context.Context, req model.Request) []issues.ConfigIssue {
	var problems []issues.ConfigIssue
	for i, spec := range req.Sources {
		if len(spec.Tags) > 0 {
			problems = append(problems, issues.ConfigIssue{Code: "unsupported-tags", Source: spec.Key().String(), Setting: fmt.Sprintf("sources[%d].tags", i), Message: "selecting skills by tags is not supported yet, remove tags"})
		}
	}
	return problems
}
