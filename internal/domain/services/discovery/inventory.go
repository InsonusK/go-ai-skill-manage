package discovery

import (
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"strings"
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// LoadFiles reads nested-file content for a skill that has already survived
// selection (tag/subpath filtering) -- deferred until here so a skill the
// filters would discard never has its nested files' bytes read at all.
func LoadFiles(s *model.Skill) error {
	for i := range s.Files {
		data, err := fs.ReadFile(s.Repo.FS, model.NestedRepoPath(s.Root, s.Files[i].Path))
		if err != nil {
			return err
		}
		s.Files[i].Data = data
	}
	return nil
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
