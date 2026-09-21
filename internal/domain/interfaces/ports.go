// Package interfaces contains the narrow outbound roles owned by the
// domain: infrastructure ports (SourceProvider, DocumentCodec, StateReader,
// PlanWriter -- their sole implementation lives in infrastructure, so the
// domain would otherwise have to import it directly) plus SourceCache,
// RepositoryLookup and SourceSelector, kept as ports even though their only
// implementation is a domain service (sourcing.Manager, discovery.Detector
// respectively), because SyncService's own orchestration tests substitute a
// fake for them. RelationExpander and SyncPlanner were removed: both had
// exactly one implementation, in another domain package, and were never
// faked -- SyncService depends on relations.Expander/planning.Planner
// directly instead.
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
