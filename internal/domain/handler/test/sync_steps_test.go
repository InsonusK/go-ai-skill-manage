package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// targetFolders is a fake interfaces.StateReader and PlanWriter: the
// folders' entries in memory, and what Apply wrote, read from the skills
// the way the real writer reads them.
type targetFolders struct {
	state   map[string]map[string]model.Managed
	failing map[string]string
	// written maps a target path to "{skill}/{file}" -> content.
	written map[string]map[string]string
	order   *[]string
}

func (t targetFolders) Snapshot(ctx context.Context, target string) (map[string]model.Managed, error) {
	if msg, ok := t.failing[target]; ok {
		return nil, errors.New(msg)
	}
	return t.state[target], nil
}

func (t targetFolders) Apply(ctx context.Context, plan entity.TargetPlan) error {
	*t.order = append(*t.order, plan.Target.Path)
	out := map[string]string{}
	for _, op := range plan.Operations {
		if op.Skill == nil {
			continue
		}
		files, err := op.Skill.Files()
		if err != nil {
			return err
		}
		for _, f := range append([]*entity.TargetFile{op.Skill.MainFile()}, files...) {
			content, err := f.Content()
			if err != nil {
				return err
			}
			out[op.Name+"/"+f.Path()] = string(content)
		}
	}
	t.written[plan.Target.Path] = out
	return nil
}

var _ interfaces.StateReader = targetFolders{}
var _ interfaces.PlanWriter = targetFolders{}

func registerSyncSteps(sc *godog.ScenarioContext, trees *map[string]fstest.MapFS, req *model.Request, problems *issues.SkillIssues) {
	var folders targetFolders
	var order []string
	var result handler.SyncResult
	var failure error

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		order, result, failure = nil, handler.SyncResult{}, nil
		folders = targetFolders{state: map[string]map[string]model.Managed{}, failing: map[string]string{}, written: map[string]map[string]string{}, order: &order}
		return ctx, nil
	})

	sc.Step(`^a target "([^"]*)" at "([^"]*)" with adapters "([^"]*)"$`, func(ctx context.Context, name, path, adapters string) error {
		req.Targets = append(req.Targets, model.Target{Name: name, Path: path, Adapters: list(adapters)})
		return nil
	})
	// the target folder holds: a table of entry name and whether it has
	// the marker.
	sc.Step(`^the target folder "([^"]*)" holds$`, func(ctx context.Context, path string, t *godog.Table) error {
		entries := map[string]model.Managed{}
		for _, row := range t.Rows[1:] {
			entries[row.Cells[0].Value] = model.Managed{Exists: true, Managed: row.Cells[1].Value == "yes"}
		}
		folders.state[path] = entries
		return nil
	})
	sc.Step(`^reading the target folder "([^"]*)" fails with "([^"]*)"$`, func(ctx context.Context, path, msg string) error {
		folders.failing[path] = msg
		return nil
	})
	sc.Step(`^it is a dry run$`, func(ctx context.Context) error {
		req.DryRun = true
		return nil
	})
	sc.Step(`^orphans are removed$`, func(ctx context.Context) error {
		req.RemoveOrphans = true
		return nil
	})

	sc.Step(`^I run sync$`, func(ctx context.Context) error {
		manager := sourcing.NewManager(map[string]interfaces.SourceProvider{"local": repositories{trees: *trees}}, "")
		result, failure = handler.SyncService{Sources: manager, State: folders, Writer: folders}.Run(ctx, *req)
		var list issues.SkillIssues
		if errors.As(failure, &list) {
			*problems = list
		}
		testsupport.Log("result=%+v error=%v", result, failure)
		return nil
	})

	sc.Step(`^sync succeeds$`, func(ctx context.Context) error { return failure })
	sc.Step(`^sync fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v; want one with %q", failure, want)
		}
		return nil
	})
	// the target issues are: a JSON list of [code, target, skill].
	sc.Step(`^the target issues are$`, func(ctx context.Context, d *godog.DocString) error {
		var list issues.TargetIssues
		if !errors.As(failure, &list) {
			return fmt.Errorf("error %v is not target issues", failure)
		}
		got := [][]string{}
		for _, i := range list {
			got = append(got, []string{i.Code, i.Target, i.Skill})
		}
		return testsupport.JSON(got, d)
	})
	sc.Step(`^the synchronized skills are "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(result.Skills, ","), want)
	})
	// the plans are: target path -> list of [action, folder name].
	sc.Step(`^the plans are$`, func(ctx context.Context, d *godog.DocString) error {
		got := map[string][][]string{}
		for _, plan := range result.Plans {
			ops := [][]string{}
			for _, op := range plan.Operations {
				ops = append(ops, []string{string(op.Action), op.Name})
			}
			got[plan.Target.Path] = ops
		}
		return testsupport.JSON(got, d)
	})
	sc.Step(`^the written targets are "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(order, ","), want)
	})
	// the target is written with: "{skill}/{file}" -> content.
	sc.Step(`^"([^"]*)" is written with$`, func(ctx context.Context, path string, d *godog.DocString) error {
		return testsupport.JSON(folders.written[path], d)
	})
	// the marker of a skill in a target lists these transformers.
	sc.Step(`^the marker of "([^"]*)" in "([^"]*)" lists transformers "([^"]*)"$`, func(ctx context.Context, skill, path, want string) error {
		var state struct{ Transformers []string }
		if err := json.Unmarshal([]byte(folders.written[path][skill+"/"+model.Marker]), &state); err != nil {
			return err
		}
		return testsupport.Equal(strings.Join(state.Transformers, ","), want)
	})
}
