// Package interfaces contains the narrow outbound ports owned by the
// domain and implemented in internal/infrastructure: SourceProvider
// (acquiring a repository), StateReader and PlanWriter (a target folder).
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
