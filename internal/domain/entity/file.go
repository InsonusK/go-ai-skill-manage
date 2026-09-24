package entity

import (
	"errors"
	"io/fs"
	"path"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// LinkSearcher finds links inside a file's content.
type LinkSearcher interface {
	SearchLinks(content string) ([]model.ParsedLink, error)
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
	links []*Link
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

// Path returns this file's path in form kind; SkillRelative starts from
// its skill's folder. FileRelative is refused: a file's path relative to
// its own folder is just its name.
func (f *File) Path(kind model.PathKind) (string, error) {
	if kind == model.FileRelative {
		return "", model.Problem("unsupported-path-kind", "a file has no path relative to itself")
	}
	p, err := model.MakePathInRepo(f.skill.Repo.RootPath, f.path, model.SkillRelative, f.skill.SkillDirPath)
	if err != nil {
		return "", err
	}
	return p.Path(kind, f.skill.SkillDirPath)
}

// Skill returns the skill that owns this file.
func (f *File) Skill() *Skill {
	return f.skill
}

// Links returns the links discovered in this file.
func (f *File) Links() ([]*Link, error) {
	if f.links == nil {
		if defaultLinkSearcher == nil {
			return nil, errors.New("entity: no LinkSearcher configured, call SetDefaultLinkSearcher first")
		}
		content, err := f.Content()
		if err != nil {
			return nil, err
		}
		parsed, err := defaultLinkSearcher.SearchLinks(string(content))
		if err != nil {
			return nil, err
		}
		links := make([]*Link, 0, len(parsed))
		for _, p := range parsed {
			link, err := MakeLink(f, p)
			if err != nil {
				return nil, err
			}
			links = append(links, link)
		}
		f.links = links
	}
	return f.links, nil
}
