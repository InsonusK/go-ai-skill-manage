package planning

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
	"path/filepath"
	"slices"
	"sort"
)

type Planner struct {
	State interfaces.StateReader
	Codec interfaces.DocumentCodec
}

func (p Planner) Plan(ctx context.Context, cat *model.SkillCatalog, sources interfaces.RepositoryLookup, req model.Request, target model.Target) (model.TargetPlan, error) {
	plan := model.TargetPlan{Target: target, Operations: []model.Operation{}}
	layout, err := BuildLayout(ctx, cat, sources)
	if err != nil {
		return plan, err
	}
	plan.Shared = layout.Shared
	prefix, err := filepath.Rel(req.Base, target.Path)
	if err != nil {
		return plan, err
	}
	destinations := map[string]string{}
	for k, v := range layout.Paths {
		destinations[k] = filepath.ToSlash(filepath.Join(prefix, filepath.FromSlash(v)))
	}
	state, err := p.State.Snapshot(ctx, target.Path)
	if err != nil {
		return plan, err
	}
	linkAdapter := slices.Contains(target.Adapters, "link-adapter")
	claudeAdapter := slices.Contains(target.Adapters, "claude-property-adapter")
	rewrite := func(f model.File) ([]byte, error) {
		if linkAdapter && len(f.Links) > 0 {
			updated, err := transform.Rewrite(string(f.Data), f.Links, destinations)
			if err != nil {
				return nil, err
			}
			return []byte(updated), nil
		}
		return f.Data, nil
	}
	wanted := map[string]bool{}
	for _, s := range cat.Skills {
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		wanted[s.Name] = true
		mainData, err := rewrite(s.MainFile)
		if err != nil {
			return plan, err
		}
		original, _ := s.Document.Properties["name"].(string)
		if claudeAdapter || original != s.Name {
			doc, err := p.Codec.Decode(mainData)
			if err != nil {
				return plan, err
			}
			doc.Properties["name"] = s.Name
			if claudeAdapter {
				doc = transform.Claude(doc)
			}
			mainData, err = p.Codec.Encode(doc)
			if err != nil {
				return plan, err
			}
		}
		files := []model.OutputFile{{Path: s.MainFile.Path, Data: mainData, Mode: s.MainFile.Mode}}
		for _, f := range s.Files {
			data, err := rewrite(f)
			if err != nil {
				return plan, err
			}
			files = append(files, model.OutputFile{Path: f.Path, Data: data, Mode: f.Mode})
		}
		hash := Fingerprint(append(append([]model.OutputFile{}, files...), layout.Shared...))
		existing := state[s.Name]
		action, reason := "create", "missing"
		if existing.Exists {
			if !existing.Managed {
				return plan, fmt.Errorf("unmanaged-target: %s", filepath.Join(target.Path, s.Name))
			}
			action, reason = "update", "content or transform changed"
			if !req.Force && existing.HasMain && existing.Hash == hash && existing.Version == model.TransformVersion {
				action, reason = "skip", "unchanged"
			}
			if req.Force {
				reason = "forced"
			}
		}
		plan.Operations = append(plan.Operations, model.Operation{Name: s.Name, Action: action, Reason: reason, Hash: hash, Files: files})
	}
	if req.RemoveOrphans {
		names := []string{}
		for name, s := range state {
			if s.Managed && !wanted[name] {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		for _, name := range names {
			plan.Operations = append(plan.Operations, model.Operation{Name: name, Action: "remove", Reason: "orphan"})
		}
	}
	return plan, nil
}
