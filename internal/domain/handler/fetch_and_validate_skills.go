// Package handler holds the command handlers: they only orchestrate calls
// to services and entities, with as little logic of their own as possible.
package handler

import (
	"context"
	"slices"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	skill_selector "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/selector"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator/validators"
)

// skillValidators checks the loaded skills, in this order.
var skillValidators = mustManager(
	validators.LinkValidator{},
	validators.SkillNameValidator{},
)

// FetchAndValidateSkills loads the skills req's sources select into a new
// SkillCatalog and checks them. It returns the catalog -- the selected
// skills plus, with req.AddRelations, the skills their links lead to --
// and every problem found; a problem doesn't stop the rest of the work.
//
// req must have passed config/validator.Validate.
//
// Порядок: сначала выбор по **всем** источникам (ссылки между источниками
// и дубли имён видны только на полном каталоге), затем валидаторы.
func FetchAndValidateSkills(ctx context.Context, sources *sourcing.Manager, req model.Request) (*sourcing.SkillCatalog, issues.SkillIssues) {
	catalog := &sourcing.SkillCatalog{
		Manager:           sources,
		AddRelations:      req.AddRelations,
		ExcludeFromChecks: excludeFromChecks(req),
	}
	selector := skill_selector.SkillSelector{SkillCatalog: catalog}
	var problems issues.SkillIssues
	for _, spec := range req.Sources {
		_, found := selector.Select(ctx, spec)
		problems = append(problems, found...)
	}
	problems = append(problems, skillValidators.Validate(ctx, catalog)...)
	return catalog, problems
}

// excludeFromChecks is, for each source's repository, the global folders
// excluded from checks followed by the source's own, each once.
//
// Пример: глобально [examples], у источника local:a -- [demo, examples]
// -> {local:a: [examples, demo]}; источник без своего списка -> только
// [examples].
func excludeFromChecks(req model.Request) map[model.SourceKey][]string {
	out := map[model.SourceKey][]string{}
	for _, spec := range req.Sources {
		folders := out[spec.Key()]
		if folders == nil {
			folders = slices.Clone(req.ExcludeFromChecks)
		}
		for _, f := range spec.ExcludeFromChecks {
			if !slices.Contains(folders, f) {
				folders = append(folders, f)
			}
		}
		out[spec.Key()] = folders
	}
	return out
}

func mustManager(list ...validators.CatalogValidator) *validator.Manager[*sourcing.SkillCatalog, issues.SkillIssue] {
	m, err := validator.NewManager(list...)
	if err != nil {
		// The list above is code, not configuration.
		panic(err)
	}
	return m
}
