package discovery

import (
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"strings"
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

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
