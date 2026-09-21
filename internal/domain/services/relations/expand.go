package relations

import (
	"context"
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"strings"
)

type Expander struct{ Detector discovery.Detector }

func (e Expander) Expand(ctx context.Context, cat *model.SkillCatalog, add bool, skip []string) error {
	processed := map[string]bool{}
	var issues model.Issues
	scan := func(s *model.Skill, f *model.File, repoPath string) {
		if !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
			return
		}
		for _, link := range links.Extract(string(f.Data)) {
			if links.Excluded(string(f.Data), f.Path, link, skip) {
				continue
			}
			resolved, err := links.Resolve(s.Repo, repoPath, link.Path, func(p string) bool { return cat.Owner(ctx, s.Repo.ID, p) != nil })
			if err == nil && cat.Owner(ctx, s.Repo.ID, resolved) == nil {
				var candidate *model.Skill
				candidate, err = e.Detector.Find(ctx, s.Repo, resolved)
				if err == nil && candidate != nil {
					if !add {
						err = model.Problem("unselected-skill", candidate.Name)
					} else if lerr := discovery.LoadFiles(candidate); lerr != nil {
						err = lerr
					} else {
						err = cat.GetOrAdd(ctx, candidate)
					}
				}
			}
			if err != nil {
				code := "link-error"
				var problem model.Issue
				if errors.As(err, &problem) {
					code = problem.Code
				}
				issues = append(issues, model.Issue{Code: code, Skill: s.Name, File: f.Path, Link: link.Raw, Message: err.Error()})
				continue
			}
			link.Target = model.OriginalKey(s.Repo.ID, resolved)
			f.Links = append(f.Links, link)
		}
	}
	for index := 0; index < len(cat.Skills); index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		s := cat.Skills[index]
		if processed[s.Key()] {
			continue
		}
		processed[s.Key()] = true
		scan(s, &s.MainFile, s.Main)
		for fi := range s.Files {
			scan(s, &s.Files[fi], model.NestedRepoPath(s.Root, s.Files[fi].Path))
		}
	}
	if len(issues) > 0 {
		return issues
	}
	return nil
}
