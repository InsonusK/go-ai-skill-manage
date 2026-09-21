package interfaces

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// SourceSelector selects the skills one source contributes, per its
// SourceSpec (subpaths/tags/name) -- implemented by discovery.Detector.
type SourceSelector interface {
	Select(context.Context, *model.Repository, model.SourceSpec) ([]*model.Skill, error)
}
