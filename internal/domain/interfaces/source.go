// Package interfaces contains the narrow outbound roles owned by the
// domain, split across files by theme (source acquisition, discovery,
// document codec, target state) rather than declared in one flat file --
// purely an organizational split, every type still lives in this one
// package and every consumer imports it exactly as before.
//
// SourceProvider, DocumentCodec and PlanWriter are genuine domain/
// infrastructure boundary ports: their sole implementation lives in
// internal/infrastructure/*, so the domain would otherwise have to import
// it directly. SourceCache, RepositoryLookup and SourceSelector are kept as
// ports even though their only implementation is itself a domain service
// (sourcing.Manager, discovery.Detector respectively), because they are
// realized by exactly one type but consumed by more than one unrelated
// domain package (SourceCache/RepositoryLookup by both domain/services and
// domain/services/planning) and are substituted with fakes in those
// packages' own orchestration tests.
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

// RepositoryLookup finds an already-acquired Repository by its ID, without
// fetching anything new -- implemented by *sourcing.Manager.
type RepositoryLookup interface {
	Lookup(context.Context, string) (*model.Repository, bool)
}
