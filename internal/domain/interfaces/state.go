package interfaces

import (
	"context"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// StateReader reads a target's managed-file state -- implemented by
// infrastructure/filesystem.Store.
type StateReader interface {
	Snapshot(context.Context, string) (map[string]model.Managed, error)
}

// PlanWriter carries out a TargetPlan in its target folder --
// implemented by infrastructure/filesystem.Store.
type PlanWriter interface {
	Apply(context.Context, entity.TargetPlan) error
}
