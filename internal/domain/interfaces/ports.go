// Package interfaces contains the narrow outbound roles owned by the
// domain: infrastructure ports (SourceProvider, SourceCache, RepositoryLookup,
// DocumentCodec, StateReader, PlanWriter) and the pipeline-stage ports
// (SourceSelector, RelationExpander, SyncPlanner) SyncService depends on
// uniformly, so its own orchestration is mockable independent of real
// discovery/relations/planning logic.
package interfaces

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// SourceProvider fetches (never caches) one Repository -- implemented by
// Local/Fetcher, used only inside a sourcing.Manager's own dispatch map.
type SourceProvider interface {
	Acquire(context.Context, model.SourceSpec, model.AcquisitionOptions) (*model.Repository, error)
}

// SourceCache is the caching front the domain depends on for source
// acquisition -- implemented by *sourcing.Manager.
type SourceCache interface {
	GetOrAdd(context.Context, model.SourceSpec, model.AcquisitionOptions) (*model.Repository, error)
}
type RepositoryLookup interface {
	Lookup(context.Context, string) (*model.Repository, bool)
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
	Select(context.Context, *model.Repository, model.SourceSpec) ([]*model.Skill, error)
}
type RelationExpander interface {
	Expand(context.Context, *model.SkillCatalog, bool, []string) error
}
type SyncPlanner interface {
	Plan(context.Context, *model.SkillCatalog, RepositoryLookup, model.Request, model.Target) (model.TargetPlan, error)
}
