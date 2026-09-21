package planning_test

import (
	"context"
	"encoding/json"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/planning"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
	"testing/fstest"
)

type stateReader map[string]model.Managed

func (s stateReader) Snapshot(context.Context, string) (map[string]model.Managed, error) {
	return s, nil
}

// repoLookup is a fake interfaces.RepositoryLookup: this fixture only ever
// acquires one repository, so any ID resolves to it.
type repoLookup struct{ repo *model.Repository }

func (r repoLookup) Lookup(context.Context, string) (*model.Repository, bool) { return r.repo, true }
func initialize(sc *godog.ScenarioContext) {
	var state string
	var force bool
	var tree fstest.MapFS
	var plan model.TargetPlan
	makeTree := func(raw map[string]string) {
		tree = fstest.MapFS{}
		for p, v := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
	}
	sc.Step(`^planning input with state "([^"]*)" and force "([^"]*)"$`, func(ctx context.Context, s, f string) error {
		state = s
		force = f == "true"
		makeTree(map[string]string{"a.skill.md": "---\nname: a\n---\n[B](b.skill.md)", "b.skill.md": "---\nname: b\n---\n"})
		testsupport.Log("state=%s force=%v", state, force)
		return nil
	})
	sc.Step(`^a planning tree$`, func(ctx context.Context, d *godog.DocString) error {
		state = "missing"
		force = false
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		makeTree(raw)
		testsupport.Log("files=%v", raw)
		return nil
	})
	runPlan := func(ctx context.Context, adapters []string) error {
		detector := discovery.Detector{Codec: document.Codec{}}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: tree}
		discovered, err := detector.Discover(ctx, repo, ".")
		if err != nil {
			return err
		}
		for _, s := range discovered {
			if err := discovery.LoadFiles(s); err != nil {
				return err
			}
		}
		cat := &model.Catalog{Skills: discovered}
		if err := (relations.Expander{Detector: detector}).Expand(ctx, cat, true, nil); err != nil {
			return err
		}
		skillMap, err := discovery.BuildSkillMap(cat)
		if err != nil {
			return err
		}
		sources := repoLookup{repo: repo}
		reader := stateReader{}
		planner := planning.Planner{State: reader, Codec: document.Codec{}}
		req := model.Request{Base: "/project", Force: force, RemoveOrphans: true}
		target := model.Target{Name: "default", Path: "/project/out", Adapters: adapters}
		first, err := planner.Plan(ctx, cat, skillMap, sources, req, target)
		if err != nil {
			return err
		}
		if state == "current" || state == "old" {
			for _, op := range first.Operations {
				v := model.TransformVersion
				if state == "old" {
					v = "old"
				}
				reader[op.Name] = model.Managed{Exists: true, Managed: true, HasMain: true, Hash: op.Hash, Version: v}
			}
		}
		if state == "orphan" || state == "unmanaged" {
			reader["old"] = model.Managed{Exists: true, Managed: state == "orphan"}
		}
		plan, err = planner.Plan(ctx, cat, skillMap, sources, req, target)
		return err
	}
	sc.Step(`^I plan a sync$`, func(ctx context.Context) error {
		return runPlan(ctx, []string{"link-adapter"})
	})
	sc.Step(`^I plan a sync without link-adapter$`, func(ctx context.Context) error {
		return runPlan(ctx, nil)
	})
	sc.Step(`^plan actions are "([^"]*)"$`, func(ctx context.Context, want string) error {
		out := []string{}
		for _, op := range plan.Operations {
			out = append(out, op.Name+":"+op.Action)
		}
		return testsupport.Equal(strings.Join(out, ","), want)
	})
	sc.Step(`^planned main text is$`, func(ctx context.Context, d *godog.DocString) error {
		return testsupport.Equal(string(plan.Operations[0].Files[0].Data), d.Content)
	})
	sc.Step(`^shared files are$`, func(ctx context.Context, d *godog.DocString) error {
		out := map[string]string{}
		for _, f := range plan.Shared {
			out[f.Path] = string(f.Data)
		}
		return testsupport.JSON(out, d)
	})
}
