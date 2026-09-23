package sourcing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// DefaultSkipFolders names top-level folders exempt from nested-skill
// validation -- they're still part of the owning skill and copied with it,
// just never walked into for a skill marker of their own (e.g. a packaged
// example app that happens to contain its own SKILL.md). Hardcoded for
// now and passed at SkillCatalog construction (SourceSpec.SkipFolders,
// read from each source's config, is not wired to this yet -- deferred,
// see AGENTS.md).
var defaultSkipFoldersInNestedChecker = []string{"examples"}

// SetDefaultLinkSearcher вызывается ОДИН раз при старте приложения
func SetSkipFoldersInNestedChecker(skipFolders []string) {
	defaultSkipFoldersInNestedChecker = skipFolders
}

// SkillCatalog is the active, path-driven source of truth for skills found
// in acquired repositories: given a source identity and a starting path,
// GetOrAddByPath acquires the Repository (via Manager, lazily, cached),
// recursively finds every skill at or below that path, validates each one
// (including nested-skill, checked by immediately warming the found
// Skill's own FilesByPath("") cache), and adds every valid one to itself.
type SkillCatalog struct {
	Manager        *Manager
	cachedSkillMap map[string]*entity.Skill
}

// GetByPath fetches key's Repository via Manager (lazily, cached by
// SourceKey), normalizes start against it (a single-file source ignores
// start entirely; otherwise start is resolved relative to the Repository's
// Root and validated), then recursively finds every skill at or below the
// normalized path inside it -- the same semantics Detector.DiscoverByPath
// has today (a directory that's a skill root becomes one Skill and is not
// searched further inside; a directory that isn't a skill root is
// searched into) -- validates each (pattern-conflict / invalid-name /
// nested-skill, exempting c.SkipFolders), and adds every valid one to the
// catalog, deduping by name per c.Conflict. Returns every valid skill
// found; invalid candidates are collected into the returned model.Issues.
func (c *SkillCatalog) GetByPath(ctx context.Context, key model.SourceKey, start string) ([]*entity.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.cachedSkillMap == nil {
		c.cachedSkillMap = map[string]*entity.Skill{}
	}
	repo, err := c.Manager.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	start, err = cleanRelative(repo, start)
	if err != nil {
		return nil, err
	}
	// A start path already resolved by an earlier call is served straight
	// from the cache -- skip normalizePath's own fs.Stat too, so a repeat
	// call for the same path costs zero additional filesystem reads.
	if skill, ok := c.cachedSkillMap[entity.GetSkillKey(repo.Key, start)]; ok {
		return []*entity.Skill{skill}, nil
	}
	start, err = normalizePath(repo, start)
	if err != nil {
		return nil, err
	}
	out := []*entity.Skill{}
	var issues model.Issues
	var scan func(string)
	scan = func(p string) {
		if err := ctx.Err(); err != nil {
			issues = append(issues, model.Issue{Code: "canceled", File: p, Message: err.Error()})
			return
		}

		//lookup in cachedSkillMap first
		if skill, ok := c.cachedSkillMap[entity.GetSkillKey(repo.Key, p)]; ok {
			out = append(out, skill)
			return
		}

		info, err := fs.Stat(repo.FS, p)
		if err != nil {
			if !isNotExist(err) {
				issues = append(issues, model.Issue{Code: "source-read", File: p, Message: err.Error()})
			}
			return
		}

		// A single-file source ignores start entirely, so the only path it can
		if !info.IsDir() {
			if strings.HasSuffix(p, ".skill.md") {
				skill, err := entity.MakeSkill(repo, p, "", entity.FlatSkill)
				if err != nil {
					issues = append(issues, model.Issue{Code: "invalid-name", File: p, Message: err.Error()})
					return
				}
				c.validateAndAdd(skill, &out, &issues)
			}
			return
		}

		// A directory that's a skill root becomes one Skill and is not searched
		skill, err := c.isSkillDir(repo, p)
		if err != nil {
			issues = append(issues, model.Issue{Code: "invalid-skill", File: p, Message: err.Error()})
			return
		}
		if skill != nil {
			c.validateAndAdd(skill, &out, &issues)
			return
		}

		entries, err := fs.ReadDir(repo.FS, p)
		if err != nil {
			issues = append(issues, model.Issue{Code: "source-read", File: p, Message: err.Error()})
			return
		}
		for _, e := range entries {
			if e.Name() != ".git" {
				scan(path.Join(p, e.Name()))
			}
		}
	}
	scan(path.Clean(start))
	if len(issues) > 0 {
		return out, issues
	}
	return out, nil
}

// accept is scan's per-candidate pipeline for a skill it just built: warm
// its FilesByPath("") cache (FilesByPath itself is a pure lister -- this is
// where the returned list is checked for a nested-skill marker not covered
// by c.SkipFolders, see nestedSkillPath), resolve name collisions via add,
// and -- once both succeed -- remember it by path and append it to out. A
// failure at either step is recorded into issues instead of the skill
// being returned; add itself only does the narrow name-collision part
// (skip/replace/reject, no file validation, no memoization, no issues
// collection -- reusable on its own terms).
//
// It also copies the warmed FilesByPath("") result into skill.Files --
// the old, eager []model.File field discovery.Rooted used to populate --
// since planning.Plan and relations.Expander still read it, not
// FilesByPath, and are out of scope for this step.
func (c *SkillCatalog) validateAndAdd(skill *entity.Skill, out *[]*entity.Skill, issues *model.Issues) {
	files, err := skill.FilesByPath("")
	if err != nil {
		*issues = append(*issues, model.Issue{Code: "source-read", Skill: skill.Name, File: skill.MainFilePath, Message: err.Error()})
		return
	}
	if nested := nestedSkillPath(files, defaultSkipFoldersInNestedChecker); nested != "" {
		*issues = append(*issues, model.Issue{Code: "nested-skill", Skill: skill.Name, File: nested, Message: fmt.Sprintf("nested-skill: %s", nested)})
		return
	}

	exist_key, is_exist := c.cachedSkillMap[skill.Key()]
	if is_exist {
		*issues = append(*issues, model.Issue{Code: "duplicate-name", Skill: skill.Name, File: skill.MainFilePath, Message: fmt.Sprintf("also defined at %s", exist_key.MainFilePath)})
		return
	}

	c.cachedSkillMap[skill.Key()] = skill
	if skill.SkillDirPath != "" {
		// A directory skill is looked up again by its own directory path,
		// not its main file's path, when a later scan() revisits the same
		// starting path -- remember it under both so that repeat lookup hits.
		c.cachedSkillMap[entity.GetSkillKey(skill.Repo.Key, skill.SkillDirPath)] = skill
	}
	*out = append(*out, skill)
}

// nestedSkillPath returns the skill-relative Path of the first file in
// files that looks like another skill's own marker (SKILL.md or
// *.skill.md) and whose top-level folder isn't one of skipFolders, or ""
// if none is found. A marker under skipFolders is deliberately not
// flagged -- that folder is still part of the owning skill and copied
// along with it (see DefaultSkipFolders), not a validation failure.
func nestedSkillPath(files []*entity.File, skipFolders []string) string {
	for _, f := range files {
		relPath, err := f.Path(entity.SkillRelative)
		if err != nil {
			continue
		}
		name := relPath
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name != "SKILL.md" && !strings.HasSuffix(name, ".skill.md") {
			continue
		}
		first := strings.SplitN(relPath, "/", 2)[0]
		exempt := false
		for _, skip := range skipFolders {
			if first == skip {
				exempt = true
				break
			}
		}
		if !exempt {
			return relPath
		}
	}
	return ""
}

// isSkillDir tests whether dir is a directory skill's root -- the shallow,
// single-level marker check only (mirrors Detector.Rooted's first
// fs.ReadDir loop); the deep nested-file walk is Skill.FilesByPath's job
// (a pure lister) and nested-skill detection is accept's own job over its
// result (nestedSkillPath), both triggered by accept once the Skill
// exists.
func (c *SkillCatalog) isSkillDir(repo *entity.Repository, dir string) (*entity.Skill, error) {
	entries, err := fs.ReadDir(repo.FS, dir)
	if err != nil {
		return nil, err
	}

	markers := []string{}
	flats := []string{}
	human := path.Base(dir)
	human = strings.TrimSuffix(human, ".skill") + ".skill.md"
	// Lookup skil files in directory, but DON'T recurse into subdirectories
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if e.Name() == "SKILL.md" || (strings.HasSuffix(dir, ".skill") && e.Name() == human) {
			markers = append(markers, e.Name())
		}
		if strings.HasSuffix(e.Name(), ".skill.md") {
			flats = append(flats, e.Name())
		}
	}
	if len(markers) == 0 {
		return nil, nil
	}
	if len(markers) > 1 {
		return nil, fmt.Errorf("pattern-conflict: multiple directory markers")
	}
	for _, flat := range flats {
		if flat != markers[0] {
			return nil, fmt.Errorf("pattern-conflict: directory marker and flat skill")
		}
	}
	main := path.Join(dir, markers[0])
	format := entity.AgentDirSkill
	if markers[0] != "SKILL.md" {
		format = entity.HumanDirSkill
	}
	return entity.MakeSkill(repo, main, dir, format)
}

// cleanRelative resolves start into a clean, valid, repo-relative path --
// pure path arithmetic, no filesystem access, so a cache hit on the result
// costs nothing beyond it.
func cleanRelative(repo *entity.Repository, start string) (string, error) {
	if filepath.IsAbs(start) {
		rel, err := filepath.Rel(repo.RootPath, start)
		if err != nil {
			return "", err
		}
		start = rel
	}
	start = filepath.ToSlash(filepath.Clean(start))
	if !fs.ValidPath(start) || strings.Contains(start, "\\") {
		return "", fmt.Errorf("unsafe subpath %q", start)
	}
	return start, nil
}

// normalizePath resolves start into a clean, valid, repo-relative path safe to
// pass to repo.FS, additionally confirming it exists.
func normalizePath(repo *entity.Repository, start string) (string, error) {
	start, err := cleanRelative(repo, start)
	if err != nil {
		return "", err
	}
	if _, err := fs.Stat(repo.FS, start); err != nil {
		return "", fmt.Errorf("subpath %q does not exist in repository %q", start, repo.Key)
	}
	return start, nil
}
