package entity

import (
	"errors"
	"io/fs"
	path_tool "path"
	"path/filepath"
	"regexp"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
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

// Skill name allow only kebab-case with lowercase letters, digits and single/double hyphens
var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-{1,2}[a-z0-9]+)*$`)

// Skill is a single skill, found in one of the three on-disk layouts (see
// SkillFormat). It is a plain, self-contained value: it knows its own
// identity and location, and lazily lists/reads its own files on request.
type Skill struct {
	// FlatSkill (a bare {name}.skill.md file, no directory): MainFilePath = "a.skill.md", SkillDirPath = "". Nothing to walk into — FilesByPath returns immediately for a flat skill.
	// AgentDirSkill (a {name}.skill/ or arbitrary directory whose marker is literally SKILL.md): MainFilePath = "a/SKILL.md", SkillDirPath = "a".
	// HumanDirSkill (a {name}.skill/ directory whose marker is {name}.skill.md, not SKILL.md): MainFilePath = "a.skill/a.skill.md", SkillDirPath = "a.skill".
	Name, MainFilePath, SkillDirPath string // MainFilePath/SkillDirPath are repo-relative; SkillDirPath is "" for FlatSkill (N/A)
	Format                           SkillFormat
	Repo                             *Repository
	Document                         SkillDocument
	MainFile                         *File // the skill's own file; Path is always "SKILL.md"

	scanned map[string][]*File // FilesByPath cache, keyed by the requested p
}

func GetSkillKey(sourceKey model.SourceKey, mainFilePath string) string {
	return sourceKey.String() + "\x00" + mainFilePath
}

// MakeSkill builds a Skill from its identity/location fields. Its MainFile
// is constructed and wired to it automatically.
func MakeSkill(repo *Repository, mainFilePath, skillDirPath string, format SkillFormat) (*Skill, error) {
	if skillDirPath != "" && !repo.IsExist(skillDirPath) {
		return nil, errors.New("skill directory does not exist in repository: " + repo.Key.String())
	}

	s := &Skill{
		MainFilePath: mainFilePath,
		SkillDirPath: skillDirPath,
		Format:       format,
		Repo:         repo,
	}
	// MainFile.path must be skill-relative like every other File (Content
	// resolves it as SkillDirPath+path) -- mainFilePath is repo-relative,
	// so it needs the same SkillDirPath-stripping FilesByPath applies to
	// nested files, not a raw pass-through.
	rel, err := filepath.Rel(filepath.FromSlash(skillDirPath), filepath.FromSlash(mainFilePath))
	if err != nil {
		return nil, err
	}

	s.MainFile = MakeFile(filepath.ToSlash(rel), s)
	skill_byte_content, err := s.MainFile.Content()
	if err != nil {
		return nil, err
	}
	s.Document, err = MakeSkillDocument(skill_byte_content)
	if err != nil {
		return nil, err
	}
	prop_name := s.Document.Properties["name"]
	if prop_name == nil {
		return nil, errors.New("skill name is missing")
	}
	s.Name = prop_name.(string)

	//Validate skill name to avoid problems with discovery and cataloging
	validName := skillNamePattern.MatchString(s.Name)
	if !validName {
		return nil, errors.New("invalid skill name: " + s.Name + " must use kebab-case with lowercase letters, digits and single/double hyphens")
	}

	return s, nil
}

func (s *Skill) Key() string { return GetSkillKey(s.Repo.Key, s.MainFilePath) }

// DirOrMarkerPath is where the skill lies in its repository: its folder,
// or its marker file for a flat skill, which has no folder.
//
// Примеры:
//   - agent-dir скил, маркер "a/guide/SKILL.md"   -> "a/guide"
//   - human-dir скил, маркер "h.skill/h.skill.md" -> "h.skill"
//   - flat-скил "b.skill.md"                      -> "b.skill.md"
//   - flat-скил "f/f.skill.md"                    -> "f/f.skill.md"
//   - скил в папке репозитория, маркер "SKILL.md" -> "."
func (s *Skill) DirOrMarkerPath() string {
	if s.SkillDirPath != "" {
		return s.SkillDirPath
	}
	return s.MainFilePath
}

// FilesByPath returns every file at and below a path resolved relative to
// the skill directory. The path may use .. to leave the skill directory,
// but its normalized repository path must remain valid inside Repo.FS.
// ("" means the skill's own root), walking only that part of the skill's
// directory -- a caller that never asks about a subtree never pays for
// walking it. It is a pure lister: it has no opinion about what it finds,
// including a file that looks like another skill's own marker -- that's a
// validation concern for the caller (see sourcing.SkillCatalog.accept),
// not this method's job. Results are cached per distinct p, so repeat
// calls with the same p don't re-walk, and returned as pointers into that
// same cached slice: Content loads its content straight into File.data, in
// place, so a later call with the same p sees it already loaded. Note
// this per-p caching means two *overlapping* p's (e.g. "" and "docs")
// each get their own walk and their own *File for the same on-disk file
// -- loading Content through one does not populate the other. Callers
// should pick one granularity for a given skill rather than mixing scopes.
// A FlatSkill has no subtree and returns immediately for any p.
func (s *Skill) FilesByPath(path string, regExFilter ...*regexp.Regexp) ([]*File, error) {
	if cached, ok := s.scanned[path]; ok {
		return filterFiles(cached, regExFilter)
	}
	if s.Format == FlatSkill {
		return nil, nil
	}
	start := s.SkillDirPath
	if path != "" {
		start = path_tool.Join(s.SkillDirPath, path)
	}
	if !fs.ValidPath(start) {
		return nil, &fs.PathError{Op: "walk", Path: start, Err: fs.ErrInvalid}
	}
	var files []*File
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
		if e.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.FromSlash(s.SkillDirPath), filepath.FromSlash(cur))
		if err != nil {
			return err
		}
		if _, err = e.Info(); err != nil {
			return err
		}
		files = append(files, MakeFile(filepath.ToSlash(rel), s))
		return nil
	})
	if err != nil {
		return files, err
	}
	if s.scanned == nil {
		s.scanned = map[string][]*File{}
	}
	s.scanned[path] = files
	return filterFiles(files, regExFilter)
}

func filterFiles(files []*File, filters []*regexp.Regexp) ([]*File, error) {
	if len(filters) == 0 || filters[0] == nil {
		return files, nil
	}
	var filtered []*File
	for _, file := range files {
		filePath, err := file.Path(model.SkillRelative)
		if err != nil {
			return nil, err
		}
		if filters[0].MatchString(filePath) {
			filtered = append(filtered, file)
		}
	}
	return filtered, nil
}
