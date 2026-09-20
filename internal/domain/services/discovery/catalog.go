package discovery

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Add resolves name collisions while growing cat, per cat.Conflict.
func Add(ctx context.Context, cat *model.Catalog, s *model.Skill) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for i, old := range cat.Skills {
		if old.Name != s.Name {
			continue
		}
		if old.Key() == s.Key() {
			return nil
		}
		if cat.Conflict == "last_wins" {
			cat.Skills[i] = s
			return nil
		}
		return model.Issue{Code: "duplicate-name", Skill: s.Name, File: s.Main, Message: fmt.Sprintf("also defined at %s", old.Main)}
	}
	cat.Skills = append(cat.Skills, s)
	return nil
}

// Owner returns the skill owning path p inside repository repoID, or nil.
func Owner(ctx context.Context, cat *model.Catalog, repoID, p string) *model.Skill {
	for _, s := range cat.Skills {
		if s.Repo.ID == repoID && Owns(s, p) {
			return s
		}
	}
	return nil
}
