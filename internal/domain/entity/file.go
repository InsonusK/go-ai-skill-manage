package entity

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// FilePathKind selects the coordinate system File.Path resolves a file's
// path in.
type FilePathKind string

const (
	// Absolute is a path relative to the filesystem root.
	Absolute FilePathKind = "absolute"
	// RepoAbsolute is a path relative to the repository folder.
	RepoAbsolute FilePathKind = "repo-relative"
	// SkillRelative is a path relative to the skill that owns the file.
	SkillRelative FilePathKind = "skill-relative"
)

// LinkSearcher finds links inside a file's content.
type LinkSearcher interface {
	SearchLinks(content string) ([]*model.Link, error)
}

// Поле приватное для пакета entity, чтобы никто снаружи не сломал его случайно
var defaultLinkSearcher LinkSearcher

// SetDefaultLinkSearcher вызывается ОДИН раз при старте приложения
func SetDefaultLinkSearcher(ls LinkSearcher) {
	defaultLinkSearcher = ls
}

// File is a single file belonging to a Skill. It is a plain, self-contained
// value: it knows its own skill-relative path and holds a back-reference to
// its owning Skill so it can resolve/read its own content lazily.
type File struct {
	data []byte

	path  string
	links []*model.Link
	skill *Skill
}

// MakeFile builds a File at skill-relative path, owned by skill.
func MakeFile(path string, skill *Skill) *File {
	return &File{path: path, skill: skill}
}

// Content returns f's own content, reading it from its owning skill's
// source repository and caching the result on f.data.
func (f *File) Content() ([]byte, error) {
	if f.data == nil {
		data, err := fs.ReadFile(f.skill.Repo.FS, path.Join(f.skill.SkillDirPath, f.path))
		if err != nil {
			return nil, err
		}
		f.data = data
	}
	return f.data, nil
}

// Path returns this file's path in the requested coordinate system.
func (f *File) Path(kind FilePathKind) (string, error) {
	switch kind {
	case SkillRelative:
		return f.path, nil
	case RepoAbsolute:
		return path.Join(f.skill.SkillDirPath, f.path), nil
	case Absolute:
		repositoryPath := path.Join(f.skill.SkillDirPath, f.path)
		return filepath.Join(f.skill.Repo.RootPath, filepath.FromSlash(repositoryPath)), nil
	default:
		return "", fmt.Errorf("unknown file path kind %q", kind)
	}
}

// Skill returns the skill that owns this file.
func (f *File) Skill() *Skill {
	return f.skill
}

// Links returns the links discovered in this file.
func (f *File) Links() ([]*model.Link, error) {
	if f.links == nil {
		content, err := f.Content()
		if err != nil {
			return nil, err
		}
		links, err := defaultLinkSearcher.SearchLinks(string(content))
		if err != nil {
			return nil, err
		}
		f.links = links
	}
	return f.links, nil
}
