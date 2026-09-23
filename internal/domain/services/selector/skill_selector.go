package skill_selector

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/tags"
)

type SkillSelector struct {
	SkillCatalog *sourcing.SkillCatalog
}

// Select resolves spec's configured subpaths (defaulting to ".") by
// calling catalog.GetOrAddByPath once per path with spec's SourceKey --
// catalog itself acquires the Repository (via its Manager, lazily/cached),
// normalizes the path (a single-file source collapses every path to that
// one file), and recursively finds/validates/adds every skill at or belowWWSS
// it. Select never acquires a Repository itself; its own job is only
// filtering catalog's results by spec's tags and applying an optional
// single-skill name override.
func (d SkillSelector) Select(ctx context.Context, spec model.SourceSpec) ([]*entity.Skill, error) {
	var issues model.Issues
	selected := []*entity.Skill{}
	seen := map[string]bool{}
	// Compile invalid filters even if discovery produces no candidates.
	if _, err := tags.Match(nil, spec.Tags); err != nil {
		return nil, err
	}
	paths := spec.Subpaths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, p := range paths {
		found, err := d.SkillCatalog.GetByPath(ctx, spec.Key(), p)
		if err != nil {
			if list, ok := err.(model.Issues); ok {
				issues = append(issues, list...)
			} else {
				issues = append(issues, issue("discovery", p, err))
			}
		}
		for _, s := range found {
			match, err := tags.Match(Tags(s), spec.Tags)
			if err != nil {
				return nil, err
			}
			if match && !seen[s.Key()] {
				selected = append(selected, s)
				seen[s.Key()] = true
			}
		}
	}
	if len(issues) > 0 {
		return selected, issues
	}
	return selected, nil
}

func issue(code, p string, err error) model.Issue {
	return model.Issue{Code: code, File: p, Message: err.Error()}
}

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

func Tags(s *entity.Skill) []string {
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
