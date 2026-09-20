package discovery

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/tags"
)

// Select applies one source's paths, filters and optional single-skill name.
func (d Detector) Select(ctx context.Context, repo *model.Repository) ([]*model.Skill, error) {
	var issues model.Issues
	selected := []*model.Skill{}
	seen := map[string]bool{}
	// Compile invalid filters even if discovery produces no candidates.
	if _, err := tags.Match(nil, repo.Spec.Tags); err != nil {
		return nil, err
	}
	for _, p := range repo.ScanPaths {
		found, err := d.Discover(ctx, repo, p)
		if err != nil {
			if list, ok := err.(model.Issues); ok {
				issues = append(issues, list...)
			} else {
				issues = append(issues, issue("discovery", p, err))
			}
		}
		for _, s := range found {
			match, err := tags.Match(Tags(s), repo.Spec.Tags)
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
	if repo.Spec.Name != "" {
		if !ValidName(repo.Spec.Name) || len(selected) != 1 {
			return nil, model.Problem("name-override", "name requires exactly one skill and a valid skill name")
		}
		selected[0].Name = repo.Spec.Name
	}
	return selected, nil
}
