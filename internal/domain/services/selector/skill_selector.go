package skill_selector

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
)

type SkillSelector struct {
	SkillCatalog *sourcing.SkillCatalog
}

// Select loads into the catalog the skills spec selects by path: for each
// of its subpaths (defaulting to ".") it calls catalog.GetOrFetchByPath
// with spec's SourceKey -- catalog itself acquires the Repository (via its
// Manager, lazily/cached), normalizes the path, and finds, validates and
// remembers every skill at or below it. Select returns them, each once.
//
// spec.Tags are not applied yet: selecting by tags is postponed (see
// AGENTS.md), every skill under the subpaths is selected.
//
// spec must have passed config/validator.Validate: a subpath leading out
// of the source reaching Select is a bug in the caller, and Select panics
// on it.
//
// Problems never stop the other subpaths, except those that stop the whole
// source: a source that can't be acquired ("source-acquire") and a
// canceled context. Every problem carries the source it belongs to.
//
// Примеры (источник local:repo, subpaths ["a", "missing", "b"]):
//   - всё есть -> скилы из "a" и "b", проблем нет
//   - нет папки "missing" -> скилы из "a" и "b" + missing-subpath (File
//     "missing")
//   - провайдер не отдал репозиторий -> ни одного скила, одна проблема
//     source-acquire (а не по одной на каждый subpath)
func (d SkillSelector) Select(ctx context.Context, spec model.SourceSpec) ([]*entity.Skill, issues.SkillIssues) {
	source := spec.Key().String()
	var problems issues.SkillIssues
	selected := []*entity.Skill{}
	seen := map[string]bool{}
	paths := spec.Subpaths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	for _, p := range paths {
		found, err := d.SkillCatalog.GetOrFetchByPath(ctx, spec.Key(), p)
		if errors.Is(err, entity.ErrSkillNotCached) {
			// Not an error to report but a broken invariant: Select is the
			// initial load, it only fetches -- following a Link to another
			// skill (Link -> entity.SkillResolver, whose cache-only lookups
			// are the only producers of ErrSkillNotCached) belongs to the later
			// validation stage, which runs after every source is selected. If
			// this fires, something reachable from GetOrFetchByPath started
			// resolving links during Select; move that call out of Select
			// rather than turning this panic into an Issue.
			panic(fmt.Sprintf("skill_selector: cache-only skill lookup reached during Select (source %s, path %q): %v", spec.Key(), p, err))
		}
		if err != nil {
			list := asIssues(err, p)
			for _, i := range list {
				if i.Code == "unsafe-subpath" {
					panic(fmt.Sprintf("skill_selector: subpath %q of source %s leads out of it, config/validator.Validate must reject it: %s", p, source, i.Message))
				}
			}
			problems = append(problems, list...)
			if stopsSource(ctx, list) {
				break
			}
		}
		for _, s := range found {
			if !seen[s.Key()] {
				selected = append(selected, s)
				seen[s.Key()] = true
			}
		}
	}
	for i := range problems {
		if problems[i].Source == "" {
			problems[i].Source = source
		}
	}
	return selected, problems
}

// asIssues turns a GetOrFetchByPath error for subpath p into issues: the
// catalog's own issues as they are, a canceled context as "canceled", any
// other error as "source-read".
func asIssues(err error, p string) issues.SkillIssues {
	var list issues.SkillIssues
	if errors.As(err, &list) {
		return list
	}
	var one issues.SkillIssue
	if errors.As(err, &one) {
		return issues.SkillIssues{one}
	}
	code := "source-read"
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = "canceled"
	}
	return issues.SkillIssues{{Code: code, File: p, Message: err.Error()}}
}

// stopsSource reports whether the remaining subpaths of the source are
// not worth trying: the context is done or the source can't be acquired.
func stopsSource(ctx context.Context, list issues.SkillIssues) bool {
	if ctx.Err() != nil {
		return true
	}
	for _, i := range list {
		if i.Code == "source-acquire" {
			return true
		}
	}
	return false
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
