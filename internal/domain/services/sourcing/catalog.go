package sourcing

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-{1,2}[a-z0-9]+)*$`)

// validName is a duplicated copy of discovery.ValidName's check -- avoids
// importing discovery, which would create a discovery <-> sourcing import
// cycle once discovery (step 3) is redesigned to call into this package.
// Flag for consolidation once discovery's own copy is retired.
func validName(name string) bool { return skillNamePattern.MatchString(name) }

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// DefaultSkipFolders names top-level folders exempt from nested-skill
// validation -- they're still part of the owning skill and copied with it,
// just never walked into for a skill marker of their own (e.g. a packaged
// example app that happens to contain its own SKILL.md). Hardcoded for
// now and passed at SkillCatalog construction (SourceSpec.SkipFolders,
// read from each source's config, is not wired to this yet -- deferred,
// see AGENTS.md).
var DefaultSkipFolders = []string{"examples"}

// SkillCatalog is the active, path-driven source of truth for skills found
// in acquired repositories: given a source identity and a starting path,
// GetOrAddByPath acquires the Repository (via Manager, lazily, cached),
// recursively finds every skill at or below that path, validates each one
// (including nested-skill, checked by immediately warming the found
// Skill's own FilesByPath("") cache), and adds every valid one to itself.
type SkillCatalog struct {
	Manager     *Manager
	Codec       interfaces.DocumentCodec
	Skills      []*model.Skill
	Conflict    string
	SkipFolders []string

	// bySource memoizes every skill GetOrAddByPath has already resolved,
	// keyed by model.OriginalKey(repoID, path) for both its Main file and
	// (for a directory skill) its Root -- so a later GetOrAddByPath call
	// that revisits the same path (e.g. an overlapping/repeated subpath)
	// reuses the already-built, already-validated *Skill instead of
	// re-reading and re-walking it. Kept in sync with Skills by
	// remember/forget (add's last_wins path forgets the replaced skill).
	bySource map[string]*model.Skill
}

// GetOrAddByPath fetches key's Repository via Manager (lazily, cached by
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
func (c *SkillCatalog) GetOrAddByPath(ctx context.Context, key model.SourceKey, start string) ([]*model.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	repo, err := c.Manager.GetOrAdd(ctx, key)
	if err != nil {
		return nil, err
	}
	start, err = normalizePath(repo, start)
	if err != nil {
		return nil, err
	}
	out := []*model.Skill{}
	var issues model.Issues
	var scan func(string)
	scan = func(p string) {
		if err := ctx.Err(); err != nil {
			issues = append(issues, model.Issue{Code: "canceled", File: p, Message: err.Error()})
			return
		}
		if skill, ok := c.bySource[model.OriginalKey(repo.ID, p)]; ok {
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
		if !info.IsDir() {
			if strings.HasSuffix(p, ".skill.md") {
				skill, err := c.makeSkill(repo, p, "", model.FlatSkill, info.Mode().Perm())
				if err != nil {
					issues = append(issues, model.Issue{Code: "invalid-name", File: p, Message: err.Error()})
					return
				}
				c.accept(skill, &out, &issues)
			}
			return
		}
		skill, err := c.rooted(repo, p)
		if err != nil {
			issues = append(issues, model.Issue{Code: "invalid-skill", File: p, Message: err.Error()})
			return
		}
		if skill != nil {
			c.accept(skill, &out, &issues)
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
func (c *SkillCatalog) accept(skill *model.Skill, out *[]*model.Skill, issues *model.Issues) {
	files, err := skill.FilesByPath("")
	if err != nil {
		*issues = append(*issues, model.Issue{Code: "source-read", Skill: skill.Name, File: skill.MainFilePath, Message: err.Error()})
		return
	}
	if nested := nestedSkillPath(files, c.SkipFolders); nested != "" {
		*issues = append(*issues, model.Issue{Code: "nested-skill", Skill: skill.Name, File: nested, Message: fmt.Sprintf("nested-skill: %s", nested)})
		return
	}
	skill.Files = make([]model.File, len(files))
	for i, f := range files {
		skill.Files[i] = *f
	}
	if err := c.add(skill); err != nil {
		if issue, ok := err.(model.Issue); ok {
			*issues = append(*issues, issue)
		} else {
			*issues = append(*issues, model.Issue{Code: "duplicate-name", Skill: skill.Name, Message: err.Error()})
		}
		return
	}
	c.remember(skill)
	*out = append(*out, skill)
}

// nestedSkillPath returns the skill-relative Path of the first file in
// files that looks like another skill's own marker (SKILL.md or
// *.skill.md) and whose top-level folder isn't one of skipFolders, or ""
// if none is found. A marker under skipFolders is deliberately not
// flagged -- that folder is still part of the owning skill and copied
// along with it (see DefaultSkipFolders), not a validation failure.
func nestedSkillPath(files []*model.File, skipFolders []string) string {
	for _, f := range files {
		name := f.Path
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		if name != "SKILL.md" && !strings.HasSuffix(name, ".skill.md") {
			continue
		}
		first := strings.SplitN(f.Path, "/", 2)[0]
		exempt := false
		for _, skip := range skipFolders {
			if first == skip {
				exempt = true
				break
			}
		}
		if !exempt {
			return f.Path
		}
	}
	return ""
}

// remember indexes s into bySource under its Main path and (for a
// directory skill) its Root, so a later GetOrAddByPath visiting either
// path again reuses s instead of re-resolving it.
func (c *SkillCatalog) remember(s *model.Skill) {
	if c.bySource == nil {
		c.bySource = map[string]*model.Skill{}
	}
	c.bySource[model.OriginalKey(s.Repo.ID, s.MainFilePath)] = s
	if s.Format != model.FlatSkill {
		c.bySource[model.OriginalKey(s.Repo.ID, s.SkillDirPath)] = s
	}
}

// forget removes s's bySource entries -- called when last_wins replaces s
// with a different skill, so a stale, no-longer-cataloged *Skill can never
// be returned by a later GetOrAddByPath call.
func (c *SkillCatalog) forget(s *model.Skill) {
	delete(c.bySource, model.OriginalKey(s.Repo.ID, s.MainFilePath))
	if s.Format != model.FlatSkill {
		delete(c.bySource, model.OriginalKey(s.Repo.ID, s.SkillDirPath))
	}
}

// rooted tests whether dir is a directory skill's root -- the shallow,
// single-level marker check only (mirrors Detector.Rooted's first
// fs.ReadDir loop); the deep nested-file walk is Skill.FilesByPath's job
// (a pure lister) and nested-skill detection is accept's own job over its
// result (nestedSkillPath), both triggered by accept once the Skill
// exists.
func (c *SkillCatalog) rooted(repo *model.Repository, dir string) (*model.Skill, error) {
	entries, err := fs.ReadDir(repo.FS, dir)
	if err != nil {
		return nil, err
	}
	markers := []string{}
	flats := []string{}
	human := path.Base(dir)
	human = strings.TrimSuffix(human, ".skill") + ".skill.md"
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
	format := model.AgentDirSkill
	if markers[0] != "SKILL.md" {
		format = model.HumanDirSkill
	}
	info, err := fs.Stat(repo.FS, main)
	if err != nil {
		return nil, err
	}
	return c.makeSkill(repo, main, dir, format, info.Mode().Perm())
}

// makeSkill reads the skill's own file, validates its frontmatter name,
// and assembles the Skill -- MainFile carries the file's bytes; nested
// files are left untouched here, handled lazily by Skill.FilesByPath.
func (c *SkillCatalog) makeSkill(repo *model.Repository, main, root string, format model.SkillFormat, mainMode fs.FileMode) (*model.Skill, error) {
	data, err := fs.ReadFile(repo.FS, main)
	if err != nil {
		return nil, err
	}
	doc, err := c.Codec.Decode(data)
	if err != nil {
		return nil, err
	}
	name, _ := doc.Properties["name"].(string)
	if !validName(name) {
		return nil, fmt.Errorf("invalid-name: %q must use lowercase letters, digits and single/double hyphens", name)
	}
	return &model.Skill{
		Name: name, MainFilePath: main, SkillDirPath: root, Format: format, Repo: repo, Document: doc,
		MainFile: model.File{Path: "SKILL.md", Data: data, Mode: mainMode},
	}, nil
}

// add resolves name collisions while growing the catalog, per c.Conflict
// -- same policy as the earlier model.SkillCatalog.GetOrAdd, minus a
// precomputed destination index (Destination scans instead).
func (c *SkillCatalog) add(s *model.Skill) error {
	for i, old := range c.Skills {
		if old.Name != s.Name {
			continue
		}
		if old.Key() == s.Key() {
			return nil
		}
		if c.Conflict == "last_wins" {
			c.forget(old)
			c.Skills[i] = s
			return nil
		}
		return model.Issue{Code: "duplicate-name", Skill: s.Name, File: s.MainFilePath, Message: fmt.Sprintf("also defined at %s", old.MainFilePath)}
	}
	c.Skills = append(c.Skills, s)
	return nil
}

// normalizePath resolves start into a clean, repo-relative path safe to
// pass to repo.FS: a single-file source (repo.SingleFile set) ignores
// start entirely and always resolves to that one file, matching Local.
// Acquire's existing single-file behavior; otherwise an absolute start is
// made relative to repo.Root, and the result is validated as a safe
// fs.FS path.
func normalizePath(repo *model.Repository, start string) (string, error) {
	if repo.SingleFile != "" {
		return repo.SingleFile, nil
	}
	if filepath.IsAbs(start) {
		rel, err := filepath.Rel(repo.Root, start)
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

// Owner returns the skill owning path p inside repository repoID, or nil.
func (c *SkillCatalog) Owner(ctx context.Context, repoID, p string) *model.Skill {
	for _, s := range c.Skills {
		if s.Repo.ID == repoID && model.OwnsPath(s.MainFilePath, s.SkillDirPath, s.Format, p) {
			return s
		}
	}
	return nil
}

// Destination resolves the output destination for a path by scanning each
// known skill via OwnsPath/RelativePath -- no precomputed index.
func (c *SkillCatalog) Destination(ctx context.Context, repoID, originalPath string) (name, dest string, ok bool) {
	for _, s := range c.Skills {
		if s.Repo.ID == repoID && model.OwnsPath(s.MainFilePath, s.SkillDirPath, s.Format, originalPath) {
			return s.Name, path.Join(s.Name, model.RelativePath(s.MainFilePath, s.SkillDirPath, originalPath)), true
		}
	}
	return "", "", false
}
