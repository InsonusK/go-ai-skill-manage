// Package interfaces contains the narrow outbound roles owned by the
// domain: infrastructure ports (SourceProvider, DocumentCodec, StateReader,
// PlanWriter) and the pipeline-stage ports (SourceSelector, RelationExpander,
// SyncPlanner) SyncService depends on uniformly, so its own orchestration is
// mockable independent of real discovery/relations/planning logic.
package interfaces

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

type SourceProvider interface {
	Acquire(context.Context, model.SourceSpec, model.AcquisitionOptions) (*model.Repository, func() error, error)
}
type DocumentCodec interface {
	Decode([]byte) (model.Document, error)
	Encode(model.Document) ([]byte, error)
}
type StateReader interface {
	Snapshot(context.Context, string) (map[string]model.Managed, error)
}
type PlanWriter interface {
	Apply(context.Context, model.TargetPlan) error
}
type SourceSelector interface {
	Select(context.Context, *model.Repository) ([]*model.Skill, error)
}
type RelationExpander interface {
	Expand(context.Context, *model.Catalog, bool, []string) error
}
type SyncPlanner interface {
	Plan(context.Context, *model.Catalog, *model.SkillMap, *model.SourceMap, model.Request, model.Target) (model.TargetPlan, error)
}
