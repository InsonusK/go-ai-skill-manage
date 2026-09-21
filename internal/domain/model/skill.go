package model

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

type File struct {
	Path  string
	Data  []byte
	Mode  fs.FileMode
	Links []Link

	skill *Skill // owning skill, for Content() to resolve Repo/Root; set by
	// FilesByPath/Find. Left nil for a File built outside this package
	// (discovery.Rooted's MainFile/s.Files construction) -- harmless for
	// MainFile (Data is always already populated, so Content() never needs
	// to resolve it), and patched on demand by the deprecated Skill.Data
	// for s.Files.
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

	scanned map[string][]*File // FilesByPath cache, keyed by the requested p
}

func (s *Skill) Key() string { return s.Repo.ID + "\x00" + s.Main }

// FileData returns the content of the skill's i-th nested file, reading it
// from the source repository and caching the result on first access --
// deferred until a caller actually needs the bytes (a link inside it is
// being scanned, or a target's plan needs its final output bytes) instead
// of loading every nested file of every skill regardless of whether
// anything ever consumes it.
func (s *Skill) FileData(i int) ([]byte, error) {
	return s.Data(&s.Files[i])
}

// FilesByPath returns the files at and below skill-relative sub-path p
// ("" means the skill's own root), walking only that part of the skill's
// directory -- a caller that never asks about a subtree never pays for
// walking it. Results are cached per distinct p, so repeat calls with the
// same p don't re-walk, and returned as pointers into that same cached
// slice: Data loads its content straight into File.Data, in place, so a
// later call with the same p sees it already loaded. Note this per-p
// caching means two *overlapping* p's (e.g. "" and "docs") each get their
// own walk and their own *File for the same on-disk file -- loading Data
// through one does not populate the other. Callers should pick one
// granularity for a given skill rather than mixing scopes.
// A FlatSkill has no subtree and returns immediately for any p. Detects
// the same nested-skill conflict Detector.Rooted's own walk checks today
// (another skill marker found inside p's subtree), scoped to whatever p
// covers -- a marker outside p's subtree is not seen by this call.
func (s *Skill) FilesByPath(p string) ([]*File, error) {
	if cached, ok := s.scanned[p]; ok {
		return cached, nil
	}
	if s.Format == FlatSkill {
		return nil, nil
	}
	start := s.Root
	if p != "" {
		start = NestedRepoPath(s.Root, p)
	}
	var files []*File
	err := fs.WalkDir(s.Repo.FS, start, func(cur string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() && e.Name() == ".git" {
			return fs.SkipDir
		}
		if cur == start || cur == s.Main {
			return nil
		}
		rel := strings.TrimPrefix(cur, s.Root+"/")
		if s.Root == "." {
			rel = cur
		}
		first := strings.Split(rel, "/")[0]
		for _, skip := range s.Repo.SkipFolders {
			if first == skip {
				if e.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if !e.IsDir() && (e.Name() == "SKILL.md" || strings.HasSuffix(e.Name(), ".skill.md")) {
			return fmt.Errorf("nested-skill: %s", cur)
		}
		if e.IsDir() || e.Name() == Marker {
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		files = append(files, &File{Path: rel, Mode: info.Mode().Perm(), skill: s})
		return nil
	})
	if s.scanned == nil {
		s.scanned = map[string][]*File{}
	}
	s.scanned[p] = files
	return files, err
}

// Find returns files under p (see FilesByPath) whose skill-relative Path
// matches re -- the same *File pointers FilesByPath(p) caches, so loading
// Data through a Find result is visible through a later FilesByPath(p)
// call with the same p too.
func (s *Skill) Find(p string, re *regexp.Regexp) ([]*File, error) {
	files, err := s.FilesByPath(p)
	if err != nil {
		return nil, err
	}
	var out []*File
	for _, f := range files {
		if re.MatchString(f.Path) {
			out = append(out, f)
		}
	}
	return out, nil
}

// Content returns f's own content, reading it from its owning skill's
// source repository and caching the result on f.Data. f must have been
// obtained from a Skill's FilesByPath/Find (or be that skill's own
// MainFile) -- a File with no owning skill wired in will panic.
func (f *File) Content() ([]byte, error) {
	if f.Data == nil {
		data, err := fs.ReadFile(f.skill.Repo.FS, NestedRepoPath(f.skill.Root, f.Path))
		if err != nil {
			return nil, err
		}
		f.Data = data
	}
	return f.Data, nil
}

// Data returns f's content, same as f.Content().
//
// Deprecated: only kept for FileData/s.Files, whose entries are built by
// discovery.Rooted (outside this package) and so never get their File.skill
// wired in by construction -- Data patches it in on the fly from the
// receiver before delegating. Once discovery constructs Files with the
// back-reference set (or s.Files is retired in favor of FilesByPath), call
// f.Content() directly and remove this method.
func (s *Skill) Data(f *File) ([]byte, error) {
	if f.skill == nil {
		f.skill = s
	}
	return f.Content()
}
