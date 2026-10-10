package transformers

import (
	"context"
	"encoding/json"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/transform"
)

// ManagedMarkerTransformer adds the marker file to every skill's folder.
// It runs last in a target's pipeline, so the transformers it lists are
// all the others. A marker the source skill already has (a target folder
// used as a source) is replaced.
//
// Пример (скил guide из "a/guide" источника github:owner/repo@main,
// коммит c0ffee, после flat и claude-when-to-use) -> "guide/.ai-skills-managed":
// {"source": {"type": "github", "path": "owner/repo", "tree": "main"},
// "commit": "c0ffee", "skill_path": "a/guide",
// "transformers": ["flat", "claude-when-to-use"], "version": "go-3"}
type ManagedMarkerTransformer struct{}

var _ transform.Transformer = ManagedMarkerTransformer{}

func (ManagedMarkerTransformer) Name() string { return "managed-marker" }

func (ManagedMarkerTransformer) Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) (issues.SkillIssues, error) {
	return nil, ManagedMarkerTransformer{}.transform(ctx, catalog)
}

func (ManagedMarkerTransformer) transform(ctx context.Context, catalog *entity.TargetSkillCatalog) error {
	applied := catalog.Applied()
	for _, s := range catalog.Skills() {
		if err := ctx.Err(); err != nil {
			return err
		}
		repo := s.Origin().Repo
		state := model.ManagedState{Source: repo.Key, Commit: repo.Commit, SkillPath: s.Origin().DirOrMarkerPath(), Transformers: applied, Version: model.TransformVersion}
		if state.Transformers == nil {
			state.Transformers = []string{}
		}
		content, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return err
		}
		content = append(content, '\n')
		files, err := s.Files()
		if err != nil {
			return err
		}
		replaced := false
		for _, f := range files {
			if f.Path() == model.Marker {
				f.SetContent(content)
				replaced = true
			}
		}
		if !replaced {
			if _, err := s.AddFile(model.Marker, content); err != nil {
				return err
			}
		}
	}
	return nil
}
