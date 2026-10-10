// Package transform turns a TargetSkillCatalog into what is written into a
// target, by running transformers over it in order (see package
// transformers for them).
package transform

import (
	"context"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
)

// Transformer changes a TargetSkillCatalog in place. An error is a failure
// to do the work (e.g. reading a file), not a problem of the skills: they
// were validated before, so a transformer that meets an invalid skill
// panics. The issues it returns are warnings: something for the user to
// check that doesn't stop the work.
type Transformer interface {
	// Name identifies the transformer in TargetSkillCatalog.Applied.
	Name() string
	Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) (issues.SkillIssues, error)
}

// Pipeline runs its transformers in the order they were given.
type Pipeline struct {
	transformers []Transformer
}

// NewPipeline makes a pipeline of transformers, in this order. A name
// given twice is refused: the applied list would be ambiguous.
func NewPipeline(transformers ...Transformer) (*Pipeline, error) {
	seen := map[string]bool{}
	for _, t := range transformers {
		if seen[t.Name()] {
			return nil, fmt.Errorf("transformer %q is given twice", t.Name())
		}
		seen[t.Name()] = true
	}
	return &Pipeline{transformers: transformers}, nil
}

// Run applies every transformer to catalog in order and records each one
// in catalog.Applied once it is done, and returns their warnings together.
// It stops at the first error -- the catalog is then half transformed and
// must not be written.
//
// Пример: конвейер [flat, marker] -> после Run catalog.Applied() =
// [..., "flat", "marker"]; если flat вернул ошибку -- marker не
// запускается, "flat" не записан.
func (p *Pipeline) Run(ctx context.Context, catalog *entity.TargetSkillCatalog) (issues.SkillIssues, error) {
	var warnings issues.SkillIssues
	for _, t := range p.transformers {
		if err := ctx.Err(); err != nil {
			return warnings, err
		}
		found, err := t.Transform(ctx, catalog)
		warnings = append(warnings, found...)
		if err != nil {
			return warnings, fmt.Errorf("transformer %s: %w", t.Name(), err)
		}
		catalog.AddApplied(t.Name())
	}
	return warnings, nil
}
