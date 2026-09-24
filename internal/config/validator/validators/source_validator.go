package validators

import (
	"context"
	"fmt"
	"path"
	"slices"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// SourceValidator checks sources that point at the same repository
// (the same SourceKey) against each other.
type SourceValidator struct{}

var _ ConfigValidator = SourceValidator{}

func (SourceValidator) Name() string { return "source-validator" }

func (SourceValidator) DependsOn() []validator.Dependency { return nil }

// Validate compares each source with the first earlier one of the same
// repository and reports on the later one:
//   - "duplicate-source": the same subpaths and tags -- it selects the same
//     skills again;
//   - "conflicting-exclude": different exclude_from_checks -- a repository's
//     skills are loaded once, so it has one list of excluded folders.
//
// Примеры (оба источника -- local "/project/skills"):
//   - subpath [a], tags [go] и subpath [./a], tags [go] -> duplicate-source
//   - subpath [a] и subpath [b]                         -> ok: разные скилы
//   - exclude_from_checks [demo] и не задан              -> conflicting-exclude
func (SourceValidator) Validate(ctx context.Context, req model.Request) []issues.ConfigIssue {
	var problems []issues.ConfigIssue
	first := map[model.SourceKey]int{}
	for i, spec := range req.Sources {
		j, seen := first[spec.Key()]
		if !seen {
			first[spec.Key()] = i
			continue
		}
		earlier := req.Sources[j]
		if slices.Equal(subpathSet(spec), subpathSet(earlier)) && slices.Equal(set(spec.Tags), set(earlier.Tags)) {
			problems = append(problems, issues.ConfigIssue{Code: "duplicate-source", Source: spec.Key().String(), Setting: fmt.Sprintf("sources[%d]", i), Message: fmt.Sprintf("same subpaths and tags as sources[%d]", j)})
		}
		if !slices.Equal(set(spec.ExcludeFromChecks), set(earlier.ExcludeFromChecks)) {
			problems = append(problems, issues.ConfigIssue{Code: "conflicting-exclude", Source: spec.Key().String(), Setting: fmt.Sprintf("sources[%d].exclude_from_checks", i), Message: fmt.Sprintf("%v differs from %v in sources[%d] of the same repository", spec.ExcludeFromChecks, earlier.ExcludeFromChecks, j)})
		}
	}
	return problems
}

// subpathSet is spec's subpaths cleaned, "." when none is set.
func subpathSet(spec model.SourceSpec) []string {
	if len(spec.Subpaths) == 0 {
		return []string{"."}
	}
	out := []string{}
	for _, p := range spec.Subpaths {
		out = append(out, path.Clean(p))
	}
	return set(out)
}

// set is values sorted and without repeats.
func set(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return slices.Compact(out)
}
