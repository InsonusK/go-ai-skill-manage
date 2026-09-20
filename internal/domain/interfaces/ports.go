// Package interfaces contains the narrow outbound roles owned by the domain.
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
