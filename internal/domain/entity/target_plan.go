package entity

import "github.com/InsonusK/go-ai-skill-manage/internal/domain/model"

// TargetAction is what a sync does with one skill folder of a target.
type TargetAction string

const (
	// CreateTarget writes a skill folder that doesn't exist yet.
	CreateTarget TargetAction = "create"
	// UpdateTarget replaces a skill folder this tool wrote before.
	UpdateTarget TargetAction = "update"
	// RemoveTarget deletes a skill folder this tool wrote before whose
	// skill is no longer synchronized.
	RemoveTarget TargetAction = "remove"
)

// TargetOperation is one planned change of a target: the skill folder Name
// and what to do with it. Skill is what to write there; nil for
// RemoveTarget. Its files are read only when the operation is applied.
type TargetOperation struct {
	Action TargetAction
	Name   string
	Skill  *TargetSkill
}

// TargetPlan is every change a sync makes in one target folder.
type TargetPlan struct {
	Target     model.Target
	Operations []TargetOperation
}
