package config

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"path/filepath"
	"strings"
)

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
	for i, a := range r.Targets {
		for _, b := range r.Targets[i+1:] {
			if overlaps(a.Path, b.Path) {
				return r, fmt.Errorf("target paths overlap: %s and %s", a.Path, b.Path)
			}
		}
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
func overlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, strings.TrimRight(b, string(filepath.Separator))+string(filepath.Separator)) || strings.HasPrefix(b, strings.TrimRight(a, string(filepath.Separator))+string(filepath.Separator))
}
