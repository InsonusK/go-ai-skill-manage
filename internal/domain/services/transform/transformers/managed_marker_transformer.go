package transformers

import (
	"context"
	"encoding/json"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
)

// ManagedState is the content of the marker file (model.Marker) that
// marks a target folder as written by this tool: only such folders are
// replaced or removed by a later sync.
type ManagedState struct {
	// Source is the source the skill comes from (model.SourceKey.String()).
	Source string `json:"source"`
	// SkillPath is where the skill lies in its source: its folder, or its
	// marker file for a flat skill (entity.Skill.DirOrMarkerPath).
	SkillPath string `json:"skill_path"`
	// Transformers are the transformers applied before the marker, in order.
	Transformers []string `json:"transformers"`
	Version      string   `json:"version"`
}

// ManagedMarkerTransformer adds the marker file to every skill's folder.
// It runs last in a target's pipeline, so the transformers it lists are
// all the others. A marker the source skill already has (a target folder
// used as a source) is replaced.
//
// Пример (скил guide из "a/guide" источника local:repo после flat и
// claude-when-to-use) -> "guide/.ai-skills-managed":
// {"source": "local:repo", "skill_path": "a/guide",
// "transformers": ["flat", "claude-when-to-use"], "version": "go-2"}
type ManagedMarkerTransformer struct{}

var _ transform.Transformer = ManagedMarkerTransformer{}

func (ManagedMarkerTransformer) Name() string { return "managed-marker" }

func (ManagedMarkerTransformer) Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) error {
	applied := catalog.Applied()
	for _, s := range catalog.Skills() {
		if err := ctx.Err(); err != nil {
			return err
		}
		state := ManagedState{Source: s.Origin().Repo.Key.String(), SkillPath: s.Origin().DirOrMarkerPath(), Transformers: applied, Version: model.TransformVersion}
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
