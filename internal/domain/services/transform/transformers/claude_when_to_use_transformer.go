package transformers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
)

// ClaudeWhenToUseTransformer renames a skill's whenToUse frontmatter
// property to when_to_use, the name Claude Code knows: it appends
// when_to_use to the description in its skill listing and silently ignores
// a property it doesn't know, whenToUse included. A list value is joined
// with ", ". A skill that already has when_to_use keeps it, and its
// whenToUse is left as is, with a warning.
//
// Примеры (frontmatter):
//   - whenToUse: "on review"         -> when_to_use: "on review"
//   - whenToUse: [review, refactor]  -> when_to_use: "review, refactor"
//   - whenToUse и when_to_use оба    -> без изменений + warning
//   - нет whenToUse                  -> файл не трогается
type ClaudeWhenToUseTransformer struct{}

var _ transform.Transformer = ClaudeWhenToUseTransformer{}

func (ClaudeWhenToUseTransformer) Name() string { return "claude-when-to-use" }

func (ClaudeWhenToUseTransformer) Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) error {
	for _, s := range catalog.Skills() {
		if err := ctx.Err(); err != nil {
			return err
		}
		doc, err := s.Document()
		if err != nil {
			return err
		}
		value, ok := doc.Properties["whenToUse"]
		if !ok || value == nil {
			continue
		}
		if _, native := doc.Properties["when_to_use"]; native {
			slog.WarnContext(ctx, "skill has both whenToUse and when_to_use, keeping when_to_use", "skill", s.Name())
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
			return err
		}
	}
	return nil
}
