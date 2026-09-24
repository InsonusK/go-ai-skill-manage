package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// SkillNameValidator checks that no two loaded skills share a name --
// across all sources, since skills are copied into targets by name.
// When link-validator is registered it must run first, so the skills it
// loads while following links are checked too.
type SkillNameValidator struct{}

var _ validator.Validator = SkillNameValidator{}

func (SkillNameValidator) Name() string { return "skill-name-validator" }

func (SkillNameValidator) DependsOn() []validator.Dependency {
	return []validator.Dependency{{Name: LinkValidator{}.Name(), IsRequired: false}}
}

// Validate reports every skill whose name another loaded skill also has,
// each with the locations of the others, in load order.
//
// Пример: скилы "guide" в "local:a" папке "guide" и в "local:b" папке
// "x/guide" -> два Issue "duplicate-name", по одному на каждый скил:
//   - Source "local:a", SkillPath "guide":   "also defined at local:b x/guide"
//   - Source "local:b", SkillPath "x/guide": "also defined at local:a guide"
func (SkillNameValidator) Validate(ctx context.Context, catalog *sourcing.SkillCatalog) model.Issues {
	skills := catalog.Skills()
	byName := map[string][]*entity.Skill{}
	for _, s := range skills {
		byName[s.Name] = append(byName[s.Name], s)
	}
	var issues model.Issues
	for _, s := range skills {
		same := byName[s.Name]
		if len(same) < 2 {
			continue
		}
		var others []string
		for _, o := range same {
			if o != s {
				others = append(others, o.Repo.Key.String()+" "+o.DirOrMarkerPath())
			}
		}
		issues = append(issues, skillIssue(s, model.Issue{Code: "duplicate-name", Message: fmt.Sprintf("also defined at %s", strings.Join(others, ", "))}))
	}
	return issues
}
