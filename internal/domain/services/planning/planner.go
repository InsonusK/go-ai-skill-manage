// Package planning decides what a sync does in a target folder before
// anything is written (see Plan).
package planning

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
)

// Plan decides what to do with each skill folder of target, without
// writing anything. It compares catalog -- the skills as they are to be
// written there, after the target's transformers -- with state, what the
// target folder holds now: every entry of it by name, and whether it is a
// folder this tool wrote (it has the marker file, model.Marker).
//
// For each skill of catalog, in catalog order, the folder named after it:
//   - doesn't exist -> CreateTarget;
//   - exists and has the marker -> UpdateTarget: the folder is replaced as
//     a whole, so files the skill no longer has go away (there is no hash
//     yet to skip a skill that didn't change);
//   - exists without the marker (someone's own skill, or any file or
//     symlink, of the same name) -> an "unmanaged-target" issue and no
//     operation: this tool never overwrites what it didn't write.
//
// Then, with removeOrphans, every folder with the marker that no skill of
// catalog has -> RemoveTarget, in name order. Entries without the marker
// that no skill has are left alone.
//
// It only decides: a caller plans every target first and writes only if
// no plan has issues, so a problem in one target leaves all of them
// untouched; a dry run just reports the plans.
//
// Пример: в каталоге guide и review; в target лежат guide/ (с маркером),
// old/ (с маркером), mine/ (без маркера):
//   - removeOrphans=true  -> [update guide, create review, remove old]
//   - removeOrphans=false -> [update guide, create review]
//   - будь review/ без маркера -> [update guide, remove old] + проблема
//     unmanaged-target для review
func Plan(target model.Target, catalog *entity.TargetSkillCatalog, state map[string]model.Managed, removeOrphans bool) (entity.TargetPlan, issues.TargetIssues) {
	plan := entity.TargetPlan{Target: target, Operations: []entity.TargetOperation{}}
	var problems issues.TargetIssues
	wanted := map[string]bool{}
	for _, s := range catalog.Skills() {
		name := s.Name()
		wanted[name] = true
		entry := state[name]
		switch {
		case !entry.Exists:
			plan.Operations = append(plan.Operations, entity.TargetOperation{Action: entity.CreateTarget, Name: name, Skill: s})
		case entry.Managed:
			plan.Operations = append(plan.Operations, entity.TargetOperation{Action: entity.UpdateTarget, Name: name, Skill: s})
		default:
			problems = append(problems, issues.TargetIssue{Code: "unmanaged-target", Target: target.Path, Skill: name, Message: fmt.Sprintf("%s exists but was not written by this tool (no %s): remove or rename it", filepath.Join(target.Path, name), model.Marker)})
		}
	}
	if removeOrphans {
		orphans := []string{}
		for name, entry := range state {
			if entry.Managed && !wanted[name] {
				orphans = append(orphans, name)
			}
		}
		sort.Strings(orphans)
		for _, name := range orphans {
			plan.Operations = append(plan.Operations, entity.TargetOperation{Action: entity.RemoveTarget, Name: name})
		}
	}
	return plan, problems
}
