package config

import (
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"path/filepath"
)

// Resolve applies overrides and makes every local path absolute from base.
// It doesn't check the result: that is config/validator.Validate's job,
// run on what Resolve returns.
func Resolve(c Config, o Overrides, base string) (model.Request, error) {
	r := c.Request
	absolute, err := filepath.Abs(base)
	if err != nil {
		return r, err
	}
	r.Base = absolute
	r.Sources = append([]model.SourceSpec{}, r.Sources...)
	r.Targets = append([]model.Target{}, r.Targets...)
	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		return filepath.Join(absolute, p)
	}
	if r.TempDir != "" {
		r.TempDir = resolve(r.TempDir)
	}
	for i := range r.Sources {
		if r.Sources[i].Type == "local" {
			r.Sources[i].Path = resolve(r.Sources[i].Path)
		}
	}
	if o.Target != "" {
		r.Targets = []model.Target{{Name: "default", Path: o.Target, Adapters: []string{"link-adapter"}}}
	}
	for i := range r.Targets {
		r.Targets[i].Path = resolve(r.Targets[i].Path)
	}
	r.DryRun = r.DryRun || o.DryRun
	r.Force = o.Force
	if o.RemoveOrphans != nil {
		r.RemoveOrphans = *o.RemoveOrphans
	}
	if o.AddRelations != nil {
		r.AddRelations = *o.AddRelations
	}
	return r, nil
}
