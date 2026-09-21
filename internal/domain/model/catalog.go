package model

import (
	"context"
	"fmt"
)

// Catalog holds the skills selected from all sources for one synchronization
// run, plus the name-collision policy applied while it grows.
type Catalog struct {
	Skills   []*Skill
	Conflict string
}

// Add resolves name collisions while growing the catalog, per c.Conflict.
func (c *Catalog) Add(ctx context.Context, s *Skill) error {
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
		return Issue{Code: "duplicate-name", Skill: s.Name, File: s.Main, Message: fmt.Sprintf("also defined at %s", old.Main)}
	}
	c.Skills = append(c.Skills, s)
	return nil
}

// Owner returns the skill owning path p inside repository repoID, or nil.
func (c *Catalog) Owner(ctx context.Context, repoID, p string) *Skill {
	for _, s := range c.Skills {
		if s.Repo.ID == repoID && OwnsPath(s.Main, s.Root, s.Format, p) {
			return s
		}
	}
	return nil
}
