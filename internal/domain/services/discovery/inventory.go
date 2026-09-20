package discovery

import (
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"path"
	"strings"
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// Files returns the immutable input bytes used by validation and output planning.
func Files(s *model.Skill) ([]model.File, error) {
	out := []model.File{}
	err := fs.WalkDir(s.Repo.FS, s.Root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			if e.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if e.Name() == model.Marker {
			return nil
		}
		data, err := fs.ReadFile(s.Repo.FS, p)
		if err != nil {
			return err
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		out = append(out, model.File{Path: p, Data: data, Mode: info.Mode().Perm()})
		return nil
	})
	return out, err
}
func Tags(s *model.Skill) []string {
	raw := s.Document.Properties["tags"]
	if text, ok := raw.(string); ok {
		return []string{strings.TrimSpace(text)}
	}
	out := []string{}
	if list, ok := raw.([]any); ok {
		for _, v := range list {
			if text, ok := v.(string); ok {
				out = append(out, strings.TrimSpace(text))
			}
		}
	}
	return out
}
func Owns(s *model.Skill, p string) bool {
	if p == s.Main || p == s.Root {
		return true
	}
	return !s.Flat && (s.Root == "." || strings.HasPrefix(p, strings.TrimSuffix(s.Root, "/")+"/"))
}
func Relative(s *model.Skill, p string) string {
	if p == s.Main || p == s.Root {
		return "SKILL.md"
	}
	if s.Root == "." {
		return p
	}
	return strings.TrimPrefix(p, path.Clean(s.Root)+"/")
}
