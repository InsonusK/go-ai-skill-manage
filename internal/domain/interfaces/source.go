// Package interfaces contains the narrow outbound roles owned by the
// domain, split across files by theme (source acquisition, discovery,
// document codec, target state) rather than declared in one flat file --
// purely an organizational split, every type still lives in this one
// package and every consumer imports it exactly as before.
//
// SourceProvider, DocumentCodec and PlanWriter are genuine domain/
// infrastructure boundary ports: their sole implementation lives in
// internal/infrastructure/*, so the domain would otherwise have to import
// it directly. RepositoryLookup is kept as a port even though its only
// implementation is itself a domain service (sourcing.Manager), because it
// is realized by exactly one type but consumed by more than one unrelated
// domain package (domain/services and domain/services/planning) and is
// substituted with a fake in those packages' own orchestration tests.
// (services.SourceSelector, the discovery-selection port, and the former
// SourceCache -- narrowed to the concrete *sourcing.Manager once it had
// exactly one consumer -- live in internal/domain/services instead, since
// their signatures need sourcing.SkillCatalog/*sourcing.Manager and
// sourcing already imports this package.)
package interfaces

import (
	"context"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// SourceProvider fetches (never caches) one Repository -- implemented by
// Local/Fetcher, used only inside a sourcing.Manager's own dispatch map.
// Takes only a SourceKey (identity: type/path/tree) -- Subpaths/Tags/Name
// are a SourceSpec's selection concern, applied later by whoever selects
// skills, not by acquisition. Folders excluded from checks
// (SourceSpec.ExcludeFromChecks) don't reach the Repository either: they
// live in sourcing.SkillCatalog.ExcludeFromChecks.
type SourceProvider interface {
	Acquire(context.Context, model.SourceKey, model.AcquisitionOptions) (*entity.Repository, error)
}

// RepositoryLookup finds an already-acquired Repository by its ID, without
// fetching anything new -- implemented by *sourcing.Manager.
type RepositoryLookup interface {
	Lookup(context.Context, string) (*entity.Repository, bool)
}
