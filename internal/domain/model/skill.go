package model

import "io/fs"

type File struct {
	Path  string
	Data  []byte
	Mode  fs.FileMode
	Links []Link
}

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

type Skill struct {
	Name, Main, Root string // Main/Root are repo-relative; Root is "" for FlatSkill (N/A)
	Format           SkillFormat
	Repo             *Repository
	Document         Document
	MainFile         File   // the skill's own file; Path is always "SKILL.md", Data always populated
	Files            []File // nested files only (never the main file); Path is skill-relative
}

func (s *Skill) Key() string { return s.Repo.ID + "\x00" + s.Main }

// FileData returns the content of the skill's i-th nested file, reading it
// from the source repository and caching the result on first access --
// deferred until a caller actually needs the bytes (a link inside it is
// being scanned, or a target's plan needs its final output bytes) instead
// of loading every nested file of every skill regardless of whether
// anything ever consumes it.
func (s *Skill) FileData(i int) ([]byte, error) {
	f := &s.Files[i]
	if f.Data == nil {
		data, err := fs.ReadFile(s.Repo.FS, NestedRepoPath(s.Root, f.Path))
		if err != nil {
			return nil, err
		}
		f.Data = data
	}
	return f.Data, nil
}
