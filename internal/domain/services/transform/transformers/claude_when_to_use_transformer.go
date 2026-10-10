package transformers

import (
	"context"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/transform"
)

// ClaudeWhenToUseTransformer renames a skill's whenToUse frontmatter
// property to when_to_use, the name Claude Code knows: it appends
// when_to_use to the description in its skill listing and silently ignores
// a property it doesn't know, whenToUse included. A list value is joined
// with ", ". A skill that already has when_to_use keeps it, and its
// whenToUse is left as is, with a "when-to-use-both" warning.
//
// Примеры (frontmatter):
//   - whenToUse: "on review"         -> when_to_use: "on review"
//   - whenToUse: [review, refactor]  -> when_to_use: "review, refactor"
//   - whenToUse и when_to_use оба    -> без изменений + warning
//   - нет whenToUse                  -> файл не трогается
type ClaudeWhenToUseTransformer struct{}

var _ transform.Transformer = ClaudeWhenToUseTransformer{}

func (ClaudeWhenToUseTransformer) Name() string { return "claude-when-to-use" }

func (ClaudeWhenToUseTransformer) Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) (issues.SkillIssues, error) {
	var warnings issues.SkillIssues
	for _, s := range catalog.Skills() {
		if err := ctx.Err(); err != nil {
			return warnings, err
		}
		doc, err := s.Document()
		if err != nil {
			return warnings, err
		}
		value, ok := doc.Properties["whenToUse"]
		if !ok || value == nil {
			continue
		}
		if _, native := doc.Properties["when_to_use"]; native {
			warning := issues.SkillIssue{Code: issues.CodeWhenToUseBoth, Skill: s.Name(), Message: "the skill has both whenToUse and when_to_use: when_to_use is kept, remove one of them"}
			if origin := s.Origin(); origin != nil {
				warning.Source, warning.SkillPath = origin.Repo.Key.String(), origin.DirOrMarkerPath()
			}
			warnings = append(warnings, warning)
			continue
		}
		if list, isList := value.([]any); isList {
			parts := []string{}
			for _, v := range list {
				parts = append(parts, fmt.Sprint(v))
			}
			value = strings.Join(parts, ", ")
		}
		doc.Properties["when_to_use"] = value
		delete(doc.Properties, "whenToUse")
		if err := s.SetDocument(doc); err != nil {
			return warnings, err
		}
	}
	return warnings, nil
}
