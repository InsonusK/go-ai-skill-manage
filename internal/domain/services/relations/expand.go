package relations

import (
	"context"
	"errors"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
)

type Expander struct{ Detector discovery.SkillSelector }

func (e Expander) Expand(ctx context.Context, cat *model.SkillCatalogImpl, add bool, skip []string) error {
	processed := map[string]bool{}
	var problems issues.SkillIssues
	scan := func(s *model.SkillImpl, f *model.FileImpl, repoPath string, data []byte) {
		if !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
			return
		}
		for _, link := range links.Extract(string(data)) {
			if links.Excluded(string(data), f.Path, link, skip) {
				continue
			}
			resolved, err := links.Resolve(s.Repo, repoPath, link.Path, func(p string) bool { return cat.Owner(ctx, s.Repo.ID, p) != nil })
			if err == nil && cat.Owner(ctx, s.Repo.ID, resolved) == nil {
				var candidate *model.SkillImpl
				candidate, err = e.Detector.Find(ctx, s.Repo, resolved)
				if err == nil && candidate != nil {
					if !add {
						err = issues.Problem("unselected-skill", candidate.Name)
					} else {
						err = cat.GetOrAdd(ctx, candidate)
					}
				}
			}
			if err != nil {
				code := "link-error"
				var problem issues.SkillIssue
				if errors.As(err, &problem) {
					code = problem.Code
				}
				problems = append(problems, issues.SkillIssue{Code: code, Skill: s.Name, File: f.Path, Link: link.Raw, Message: err.Error()})
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
		scan(s, &s.MainFile, s.MainFilePath, s.MainFile.Data)
		for fi := range s.Files {
			f := &s.Files[fi]
			if !strings.HasSuffix(strings.ToLower(f.Path), ".md") {
				continue
			}
			data, err := s.FileData(fi)
			if err != nil {
				problems = append(problems, issues.SkillIssue{Code: "source-read", Skill: s.Name, File: f.Path, Message: err.Error()})
				continue
			}
			scan(s, f, model.NestedRepoPath(s.SkillDirPath, f.Path), data)
		}
	}
	if len(problems) > 0 {
		return problems
	}
	return nil
}
