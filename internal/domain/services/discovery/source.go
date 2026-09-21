package discovery

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/tags"
)

// scanPaths resolves the roots to scan for one source's selection: a single
// file source always scans just that file, regardless of configured
// subpaths (matching the previous behavior baked into Local.Acquire).
func scanPaths(repo *model.Repository, subpaths []string) ([]string, error) {
	if repo.SingleFile != "" {
		return []string{repo.SingleFile}, nil
	}
	paths := subpaths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	normalized := make([]string, 0, len(paths))
	for _, p := range paths {
		if filepath.IsAbs(p) {
			rel, err := filepath.Rel(repo.Root, p)
			if err != nil {
				return nil, err
			}
			p = rel
		}
		p = filepath.ToSlash(filepath.Clean(p))
		if !fs.ValidPath(p) || strings.Contains(p, "\\") {
			return nil, fmt.Errorf("unsafe subpath %q", p)
		}
		normalized = append(normalized, p)
	}
	return normalized, nil
}

// Select applies one source's paths, filters and optional single-skill name.
func (d Detector) Select(ctx context.Context, repo *model.Repository, spec model.SourceSpec) ([]*model.Skill, error) {
	var issues model.Issues
	selected := []*model.Skill{}
	seen := map[string]bool{}
	// Compile invalid filters even if discovery produces no candidates.
	if _, err := tags.Match(nil, spec.Tags); err != nil {
		return nil, err
	}
	paths, err := scanPaths(repo, spec.Subpaths)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		found, err := d.DiscoverByPath(ctx, repo, p)
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
	if spec.Name != "" {
		if !ValidName(spec.Name) || len(selected) != 1 {
			return nil, model.Problem("name-override", "name requires exactly one skill and a valid skill name")
		}
		selected[0].Name = spec.Name
	}
	return selected, nil
}
