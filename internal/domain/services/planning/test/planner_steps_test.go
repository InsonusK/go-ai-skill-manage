package planning_test

import (
	"context"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func initialize(sc *godog.ScenarioContext) {
	var catalog *entity.TargetSkillCatalog
	var state map[string]model.Managed
	var removeOrphans bool
	var plan entity.TargetPlan
	var problems issues.TargetIssues

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		catalog, state, removeOrphans = entity.NewTargetSkillCatalog(nil), map[string]model.Managed{}, false
		return ctx, nil
	})
	// skills to write: comma-separated names, each an agent-dir skill.
	sc.Step(`^the skills to write are "([^"]*)"$`, func(ctx context.Context, names string) error {
		tree := fstest.MapFS{}
		repo := &entity.Repository{Key: model.SourceKey{Type: "local", Path: "repo"}, FS: tree}
		var skills []*entity.Skill
		for _, n := range strings.Split(names, ",") {
			tree[n+"/SKILL.md"] = &fstest.MapFile{Data: []byte("---\nname: " + n + "\n---\n")}
			s, err := entity.MakeSkill(repo, n+"/SKILL.md", n, entity.AgentDirSkill)
			if err != nil {
				return err
			}
			skills = append(skills, s)
		}
		catalog = entity.NewTargetSkillCatalog(skills)
		return nil
	})
	// the target holds: a table of entry name and whether it has the marker.
	sc.Step(`^the target holds$`, func(ctx context.Context, t *godog.Table) error {
		for _, row := range t.Rows[1:] {
			state[row.Cells[0].Value] = model.Managed{Exists: true, Managed: row.Cells[1].Value == "yes"}
		}
		return nil
	})
	sc.Step(`^orphans are (removed|kept)$`, func(ctx context.Context, v string) error {
		removeOrphans = v == "removed"
		return nil
	})
	sc.Step(`^I plan the target$`, func(ctx context.Context) error {
		plan, problems = planning.Plan(model.Target{Name: "claude", Path: "/p/.claude/skills"}, catalog, state, removeOrphans)
		testsupport.Log("plan=%v issues=%v", plan.Operations, problems)
		return nil
	})
	// the operations are: a JSON list of [action, name, skill written].
	sc.Step(`^the operations are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, op := range plan.Operations {
			skill := ""
			if op.Skill != nil {
				skill = op.Skill.Name()
			}
			got = append(got, []string{string(op.Action), op.Name, skill})
		}
		return testsupport.JSON(got, d)
	})
	// the plan issues are: a JSON list of [code, target, skill].
	sc.Step(`^the plan issues are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			got = append(got, []string{i.Code, i.Target, i.Skill})
		}
		return testsupport.JSON(got, d)
	})
}
