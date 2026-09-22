package interfaces

import (
	"regexp"
)

// SkillFormat names which of the three on-disk skill layouts a Skill was
// found in.
type SkillFormat string

const (
	// FlatSkill is a single "{name}.skill.md" file with no directory.
	FlatSkill SkillFormat = "flat"
	// HumanDirSkill is a "{name}.skill/" directory whose marker file is
	// "{name}.skill.md".
	HumanDirSkill SkillFormat = "human-dir"
	// AgentDirSkill is a "{name}.skill/" directory whose marker file is
	// "SKILL.md".
	AgentDirSkill SkillFormat = "agent-dir"
)

type Document struct {
	Properties     map[string]any
	Metadata       map[string]any
	Body           string
	HasFrontmatter bool
}

type SkillMetadata struct {
	// FlatSkill (a bare {name}.skill.md file, no directory): Main = "a.skill.md", Root = "". Nothing to walk into — FilesByPath returns immediately for a flat skill.
	// AgentDirSkill (a {name}.skill/ or arbitrary directory whose marker is literally SKILL.md): Main = "a/SKILL.md", Root = "a".
	// HumanDirSkill (a {name}.skill/ directory whose marker is {name}.skill.md, not SKILL.md): Main = "a.skill/a.skill.md", Root = "a.skill".
	Name, MainFilePath, SkillDirPath string // Main/Root are repo-relative; Root is "" for FlatSkill (N/A)
	Format                           SkillFormat
	Repo                             *Repository
	Document                         Document
	MainFile                         File
}

type Skill interface {
	//Get skill's unique key
	Key() string
	//Get skill's name
	Name() string
	Document() Document
	//Get files under path p (relative to skill root)
	//-- if p is empty, returns all files under skill root.
	//Caches results for repeated calls with the same p.
	FilesByPath(path string, regExFilter ...*regexp.Regexp) ([]*File, error)
}
