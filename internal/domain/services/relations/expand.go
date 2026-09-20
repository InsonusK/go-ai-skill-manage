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

func (e Expander) Expand(ctx context.Context, cat *model.Catalog, add bool, skip []string) error {
	processed := map[string]bool{}
	var issues model.Issues
	for index := 0; index < len(cat.Skills); index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		s := cat.Skills[index]
		if processed[s.Key()] {
			continue
		}
		processed[s.Key()] = true
		files, err := discovery.Files(s)
		if err != nil {
			issues = append(issues, model.Issue{Code: "source-read", Skill: s.Name, Message: err.Error()})
			continue
		}
		s.Files = files
		for fi := range s.Files {
			f := &s.Files[fi]
			if !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
				continue
			}
			for _, link := range links.Extract(string(f.Data)) {
				if links.Excluded(string(f.Data), f.Path, link, skip) {
					continue
				}
				resolved, err := links.Resolve(s.Repo, f.Path, link.Path, func(p string) bool { return cat.Owner(ctx, s.Repo.ID, p) != nil })
				if err == nil && cat.Owner(ctx, s.Repo.ID, resolved) == nil {
					var candidate *model.Skill
					candidate, err = e.Detector.Find(ctx, s.Repo, resolved)
					if err == nil && candidate != nil {
						if !add {
							err = model.Problem("unselected-skill", candidate.Name)
						} else {
							err = cat.Add(ctx, candidate)
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
				link.Target = s.Repo.ID + "\x00" + resolved
				f.Links = append(f.Links, link)
			}
		}
	}
	if len(issues) > 0 {
		return issues
	}
	return nil
}
