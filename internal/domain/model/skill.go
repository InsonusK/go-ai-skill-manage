package model

import (
	"io/fs"
	"regexp"
	"strings"
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

type SkillImpl struct {
	// FlatSkill (a bare {name}.skill.md file, no directory): Main = "a.skill.md", Root = "". Nothing to walk into — FilesByPath returns immediately for a flat skill.
	// AgentDirSkill (a {name}.skill/ or arbitrary directory whose marker is literally SKILL.md): Main = "a/SKILL.md", Root = "a".
	// HumanDirSkill (a {name}.skill/ directory whose marker is {name}.skill.md, not SKILL.md): Main = "a.skill/a.skill.md", Root = "a.skill".
	Name, MainFilePath, SkillDirPath string // Main/Root are repo-relative; Root is "" for FlatSkill (N/A)
	Format                           SkillFormat
	Repo                             *Repository
	Document                         Document
	MainFile                         FileImpl   // the skill's own file; Path is always "SKILL.md", Data always populated
	Files                            []FileImpl // nested files only (never the main file); Path is skill-relative

	scanned map[string][]*FileImpl // FilesByPath cache, keyed by the requested p
}

func (s *SkillImpl) Key() string { return s.Repo.ID + "\x00" + s.MainFilePath }

// FileData returns the content of the skill's i-th nested file, reading it
// from the source repository and caching the result on first access --
// deferred until a caller actually needs the bytes (a link inside it is
// being scanned, or a target's plan needs its final output bytes) instead
// of loading every nested file of every skill regardless of whether
// anything ever consumes it.
func (s *SkillImpl) FileData(i int) ([]byte, error) {
	return s.Data(&s.Files[i])
}

// FilesByPath returns every file at and below skill-relative sub-path p
// ("" means the skill's own root), walking only that part of the skill's
// directory -- a caller that never asks about a subtree never pays for
// walking it. It is a pure lister: it has no opinion about what it finds,
// including a file that looks like another skill's own marker -- that's a
// validation concern for the caller (see sourcing.SkillCatalog.accept),
// not this method's job. Results are cached per distinct p, so repeat
// calls with the same p don't re-walk, and returned as pointers into that
// same cached slice: Data loads its content straight into File.Data, in
// place, so a later call with the same p sees it already loaded. Note
// this per-p caching means two *overlapping* p's (e.g. "" and "docs")
// each get their own walk and their own *File for the same on-disk file
// -- loading Data through one does not populate the other. Callers should
// pick one granularity for a given skill rather than mixing scopes.
// A FlatSkill has no subtree and returns immediately for any p.
func (s *SkillImpl) FilesByPath(p string) ([]*FileImpl, error) {
	if cached, ok := s.scanned[p]; ok {
		return cached, nil
	}
	if s.Format == FlatSkill {
		return nil, nil
	}
	start := s.SkillDirPath
	if p != "" {
		start = NestedRepoPath(s.SkillDirPath, p)
	}
	var files []*FileImpl
	err := fs.WalkDir(s.Repo.FS, start, func(cur string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() && e.Name() == ".git" {
			return fs.SkipDir
		}
		if cur == start || cur == s.MainFilePath {
			return nil
		}
		if e.IsDir() || e.Name() == Marker {
			return nil
		}
		rel := strings.TrimPrefix(cur, s.SkillDirPath+"/")
		if s.SkillDirPath == "." {
			rel = cur
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		files = append(files, &FileImpl{Path: rel, Mode: info.Mode().Perm(), skill: s})
		return nil
	})
	if s.scanned == nil {
		s.scanned = map[string][]*FileImpl{}
	}
	s.scanned[p] = files
	return files, err
}

// Find returns files under p (see FilesByPath) whose skill-relative Path
// matches re -- the same *File pointers FilesByPath(p) caches, so loading
// Data through a Find result is visible through a later FilesByPath(p)
// call with the same p too.
func (s *SkillImpl) Find(p string, re *regexp.Regexp) ([]*FileImpl, error) {
	files, err := s.FilesByPath(p)
	if err != nil {
		return nil, err
	}
	var out []*FileImpl
	for _, f := range files {
		if re.MatchString(f.Path) {
			out = append(out, f)
		}
	}
	return out, nil
}

// Data returns f's content, same as f.Content().
//
// Deprecated: only kept for FileData/s.Files, whose entries are built by
// discovery.Rooted (outside this package) and so never get their File.skill
// wired in by construction -- Data patches it in on the fly from the
// receiver before delegating. Once discovery constructs Files with the
// back-reference set (or s.Files is retired in favor of FilesByPath), call
// f.Content() directly and remove this method.
func (s *SkillImpl) Data(f *FileImpl) ([]byte, error) {
	if f.skill == nil {
		f.skill = s
	}
	return f.Content()
}
