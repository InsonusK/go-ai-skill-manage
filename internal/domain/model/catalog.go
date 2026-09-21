package model

import (
	"context"
	"fmt"
	"path"
)

// SkillCatalog holds the skills selected from all sources for one
// synchronization run, the name-collision policy applied while it grows,
// and the output destination of every original file each skill owns --
// indexed incrementally as skills are added, so no separate finalization
// pass is needed once relation expansion stops growing it.
type SkillCatalog struct {
	Skills   []*Skill
	Conflict string
	dest     map[string]catalogDest
}

type catalogDest struct {
	name string
	path string
}

// OriginalKey builds the lookup key SkillCatalog, Link.Target, and a
// target's resolved output layout all use: a NUL-separated (repoID, path)
// pair, so identical relative paths from different sources never collide.
func OriginalKey(repoID, path string) string { return repoID + "\x00" + path }

// GetOrAdd resolves name collisions while growing the catalog, per
// c.Conflict, and indexes the skill's canonical output destinations.
func (c *SkillCatalog) GetOrAdd(ctx context.Context, s *Skill) error {
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
			c.deindex(old)
			c.Skills[i] = s
			c.index(s)
			return nil
		}
		return Issue{Code: "duplicate-name", Skill: s.Name, File: s.Main, Message: fmt.Sprintf("also defined at %s", old.Main)}
	}
	c.Skills = append(c.Skills, s)
	c.index(s)
	return nil
}

// index registers every original path s owns against its output
// destination: the main file, every nested file, and -- for a directory
// skill -- the root itself (a link to the directory resolves to SKILL.md).
func (c *SkillCatalog) index(s *Skill) {
	if c.dest == nil {
		c.dest = map[string]catalogDest{}
	}
	set := func(original, dest string) {
		c.dest[OriginalKey(s.Repo.ID, original)] = catalogDest{name: s.Name, path: dest}
	}
	set(s.Main, path.Join(s.Name, "SKILL.md"))
	for _, f := range s.Files {
		set(NestedRepoPath(s.Root, f.Path), path.Join(s.Name, f.Path))
	}
	if s.Format != FlatSkill {
		set(s.Root, path.Join(s.Name, "SKILL.md"))
	}
}

func (c *SkillCatalog) deindex(s *Skill) {
	del := func(original string) { delete(c.dest, OriginalKey(s.Repo.ID, original)) }
	del(s.Main)
	for _, f := range s.Files {
		del(NestedRepoPath(s.Root, f.Path))
	}
	if s.Format != FlatSkill {
		del(s.Root)
	}
}

// Owner returns the skill owning path p inside repository repoID, or nil.
func (c *SkillCatalog) Owner(ctx context.Context, repoID, p string) *Skill {
	for _, s := range c.Skills {
		if s.Repo.ID == repoID && OwnsPath(s.Main, s.Root, s.Format, p) {
			return s
		}
	}
	return nil
}

// Destination returns the output destination for the file at originalPath
// inside repoID's source, and the name of the skill owning it. It first
// tries an exact indexed-path match, then falls back to OwnsPath/
// RelativePath against each skill's Root/Main/Format, so callers get one
// answer regardless of whether the path was ever inventoried.
func (c *SkillCatalog) Destination(repoID, originalPath string) (name, dest string, ok bool) {
	if d, ok := c.dest[OriginalKey(repoID, originalPath)]; ok {
		return d.name, d.path, true
	}
	for _, s := range c.Skills {
		if s.Repo.ID == repoID && OwnsPath(s.Main, s.Root, s.Format, originalPath) {
			return s.Name, path.Join(s.Name, RelativePath(s.Main, s.Root, originalPath)), true
		}
	}
	return "", "", false
}

// Destinations returns a defensive copy of every OriginalKey -> destination
// pairing, so callers may extend their own copy (e.g. with per-target
// external-attachment destinations) without mutating the shared catalog.
func (c *SkillCatalog) Destinations() map[string]string {
	out := make(map[string]string, len(c.dest))
	for k, d := range c.dest {
		out[k] = d.path
	}
	return out
}
