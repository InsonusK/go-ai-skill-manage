package discovery

import (
	"context"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/tags"
)

// Select resolves spec's configured subpaths (defaulting to ".") by
// calling catalog.GetOrAddByPath once per path with spec's SourceKey --
// catalog itself acquires the Repository (via its Manager, lazily/cached),
// normalizes the path (a single-file source collapses every path to that
// one file), and recursively finds/validates/adds every skill at or below
// it. Select never acquires a Repository itself; its own job is only
// filtering catalog's results by spec's tags and applying an optional
// single-skill name override.
func (d Detector) Select(ctx context.Context, catalog *sourcing.SkillCatalog, spec model.SourceSpec) ([]*model.SkillImpl, error) {
	var issues model.Issues
	selected := []*model.SkillImpl{}
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
		found, err := catalog.GetOrAddByPath(ctx, spec.Key(), p)
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
