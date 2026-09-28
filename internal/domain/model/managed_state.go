package model

// ManagedState is the content of the marker file (Marker) that marks a
// target folder as written by this tool: only such folders are replaced or
// removed by a later sync. It is also the way back from a written skill to
// the source it comes from.
type ManagedState struct {
	// Source is the source the skill comes from.
	Source SourceKey `json:"source"`
	// Commit is the exact commit the skill was taken from; empty when the
	// provider doesn't know it (a local source).
	Commit string `json:"commit,omitempty"`
	// SkillPath is where the skill lies in its source: its folder, or its
	// marker file for a flat skill (entity.Skill.DirOrMarkerPath).
	SkillPath string `json:"skill_path"`
	// Transformers are the transformers applied before the marker, in order.
	Transformers []string `json:"transformers"`
	Version      string   `json:"version"`
}
