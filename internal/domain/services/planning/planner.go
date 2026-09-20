package planning

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
	"path/filepath"
	"sort"
)

type Planner struct {
	State interfaces.StateReader
	Codec interfaces.DocumentCodec
}

func (p Planner) Plan(ctx context.Context, cat *model.Catalog, skills *model.SkillMap, sources *model.SourceMap, req model.Request, target model.Target) (model.TargetPlan, error) {
	plan := model.TargetPlan{Target: target, Operations: []model.Operation{}}
	layout, err := BuildLayout(ctx, cat, skills, sources)
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
	wanted := map[string]bool{}
	for _, s := range cat.Skills {
		if err := ctx.Err(); err != nil {
			return plan, err
		}
		wanted[s.Name] = true
		files := []model.OutputFile{}
		for _, f := range s.Files {
			data := f.Data
			if len(f.Links) > 0 {
				updated, err := transform.Rewrite(string(data), f.Links, destinations)
				if err != nil {
					return plan, err
				}
				data = []byte(updated)
			}
			if f.Path == s.Main {
				claude := false
				for _, a := range target.Adapters {
					if a == "claude-property-adapter" {
						claude = true
					}
				}
				original, _ := s.Document.Properties["name"].(string)
				if claude || original != s.Name {
					doc, err := p.Codec.Decode(data)
					if err != nil {
						return plan, err
					}
					doc.Properties["name"] = s.Name
					if claude {
						doc = transform.Claude(doc)
					}
					data, err = p.Codec.Encode(doc)
					if err != nil {
						return plan, err
					}
				}
			}
			files = append(files, model.OutputFile{Path: discovery.Relative(s, f.Path), Data: data, Mode: f.Mode})
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
