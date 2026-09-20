package discovery

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

type Catalog struct {
	Skills   []*model.Skill
	Conflict string
}

func (c *Catalog) Add(ctx context.Context, s *model.Skill) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, old := range c.Skills {
		if old.Name != s.Name {
			continue
		}
		if old.Key() == s.Key() {
			return nil
		}
		if c.Conflict == "last_wins" {
			c.Skills[i] = s
			return nil
		}
		return model.Issue{Code: "duplicate-name", Skill: s.Name, File: s.Main, Message: fmt.Sprintf("also defined at %s", old.Main)}
	}
	c.Skills = append(c.Skills, s)
	return nil
}
func (c *Catalog) Owner(ctx context.Context, repoID, p string) *model.Skill {
	for _, s := range c.Skills {
		if s.Repo.ID == repoID && Owns(s, p) {
			return s
		}
	}
	return nil
}
