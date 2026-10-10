// Package transformers holds the transformers of a TargetSkillCatalog (see
// package transform).
package transformers

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/services/transform"
)

// FlatTransformer lays the skills out the way a target holds them: each
// skill in a folder named after it, its marker file -- whatever its format
// -- as SKILL.md, its other files where they are in its folder. Links in
// markdown files are rewritten to lead to the same files at their new
// places, and wikilinks become markdown links.
//
// A link to a shared file of the source that no skill holds (e.g. a
// catalog's registry entry) gets that file copied into the first skill, in
// catalog order, that links to it -- at "{name}/files/<its path in the
// source>" -- and every link to it leads there. Its own links are copied as
// written.
//
// It must run first, on the base catalog: it rewrites links using their
// positions in the source files, so a file changed before it is a bug
// (panic). A link it can't resolve is a bug too -- the skills were
// validated.
//
// Примеры (скилы guide в "a/b/guide", human-dir h в "h.skill", flat f в
// "f.skill.md"):
//   - "a/b/guide/SKILL.md", "a/b/guide/docs/x.md" -> "guide/SKILL.md",
//     "guide/docs/x.md"; "h.skill/h.skill.md" -> "h/SKILL.md";
//     "f.skill.md" -> "f/SKILL.md"
//   - в guide/docs/x.md: [h](../../../../h.skill/h.skill.md#top) ->
//     [h](../../h/SKILL.md#top)
//   - [[h.skill/h.skill.md|H]] -> [H](../h/SKILL.md)
//   - guide ссылается на "a/registry/x.md" (не в скиле) -> файл копируется в
//     "guide/files/a/registry/x.md", ссылка h на него -> ../guide/files/a/registry/x.md
type FlatTransformer struct {
	// Catalog is the loaded skill catalog the target catalog was made
	// from; links in files it excludes from checks are left as written.
	Catalog *sourcing.SkillCatalog
}

var _ transform.Transformer = FlatTransformer{}

func (FlatTransformer) Name() string { return "flat" }

func (t FlatTransformer) Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) (issues.SkillIssues, error) {
	return nil, t.transform(ctx, catalog)
}

func (t FlatTransformer) transform(ctx context.Context, catalog *entity.TargetSkillCatalog) error {
	skills := catalog.Skills()
	places := placesOf(skills)
	for _, s := range skills {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := t.rewriteLinks(ctx, s, places); err != nil {
			return err
		}
	}
	for _, s := range skills {
		name := s.Name()
		s.SetSkillDirPath(name)
		s.SetMainFilePath(name + "/SKILL.md")
		s.SetFormat(entity.AgentDirSkill)
		s.MainFile().SetPath("SKILL.md")
	}
	return nil
}

// rewriteLinks rewrites the links of s's markdown files for the new
// layout. Its files are still at their source places.
func (t FlatTransformer) rewriteLinks(ctx context.Context, s *entity.TargetSkill, places *places) error {
	files, err := s.Files()
	if err != nil {
		return err
	}
	for _, f := range append([]*entity.TargetFile{s.MainFile()}, files...) {
		rel := f.Path()
		if f.Origin() == nil || !strings.HasSuffix(strings.ToLower(rel), ".md") || t.Catalog.IsExcludedFromChecks(s.Origin(), rel) {
			continue
		}
		if f.Changed() {
			panic(fmt.Sprintf("transformers: %s of skill %s changed before the flat transformer, which must run first", rel, s.Name()))
		}
		newPlace := path.Join(s.Name(), rel)
		if f == s.MainFile() {
			newPlace = path.Join(s.Name(), "SKILL.md")
		}
		content, err := f.Content()
		if err != nil {
			return err
		}
		links, err := f.Origin().Links()
		if err != nil {
			return err
		}
		edits := []edit{}
		for _, l := range links {
			e, ok, err := rewrite(ctx, l, path.Dir(newPlace), places, s)
			if err != nil {
				return err
			}
			if ok {
				edits = append(edits, e)
			}
		}
		if len(edits) > 0 {
			f.SetContent(apply(content, edits))
		}
	}
	return nil
}

// edit replaces content[start:end] with text.
type edit struct {
	start, end int
	text       string
}

// rewrite returns the edit that makes link l, written in a file that moves
// into folder fromDir, lead to its target's new place; false when l stays
// as written (a web link, a markdown link to the same place, or one with
// only an anchor).
func rewrite(ctx context.Context, l *entity.Link, fromDir string, places *places, s *entity.TargetSkill) (edit, bool, error) {
	if l.External {
		return edit{}, false, nil
	}
	target := l.Fragment
	if l.WrittenPath != "" {
		repoPath, err := l.Path(ctx, model.RepoAbsolute, nil)
		if err != nil {
			panic(fmt.Sprintf("transformers: link %s in skill %s can't be resolved after validation: %v", l.Raw, s.Name(), err))
		}
		to, ok := places.of(s.Origin().Repo.Key, repoPath)
		if !ok {
			if to, err = places.shared(s, repoPath); err != nil {
				return edit{}, false, err
			}
		}
		target = relative(fromDir, to) + l.Fragment
	}
	if l.Format == "markdown" {
		// Only the "(path#fragment)" part changes; the rest stays as written.
		start := l.End - len(l.WrittenPath) - len(l.Fragment) - 1
		if l.Raw[len(l.Raw)-1-len(l.WrittenPath)-len(l.Fragment):len(l.Raw)-1] == target {
			return edit{}, false, nil
		}
		return edit{start: start, end: l.End - 1, text: target}, true, nil
	}
	text := l.Text
	if text == "" {
		text = strings.TrimPrefix(l.Fragment, "#")
	}
	image := ""
	if l.Image {
		image = "!"
	}
	return edit{start: l.Start, end: l.End, text: image + "[" + text + "](" + target + ")"}, true, nil
}

// relative is the path from folder fromDir to to, as a file-relative link
// path: "./x" or "../x".
func relative(fromDir, to string) string {
	rel, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(to))
	if err != nil {
		panic(fmt.Sprintf("transformers: no relative path from %s to %s: %v", fromDir, to, err))
	}
	rel = filepath.ToSlash(rel)
	switch {
	case rel == ".":
		return "./"
	case rel == ".." || strings.HasPrefix(rel, "../"):
		return rel
	}
	return "./" + rel
}

// apply makes the edits, sorted by position, to content.
func apply(content []byte, edits []edit) []byte {
	slices.SortFunc(edits, func(a, b edit) int { return a.start - b.start })
	var out []byte
	cursor := 0
	for _, e := range edits {
		out = append(out, content[cursor:e.start]...)
		out = append(out, e.text...)
		cursor = e.end
	}
	return append(out, content[cursor:]...)
}

// places tells where a path of a source repository goes in the target:
// into the skill that holds it, or, for a shared file no skill holds, into
// the skill that linked to it first.
type places struct {
	skills map[model.SourceKey][]*entity.TargetSkill
	// sharedAt maps entity.GetSkillKey(source, path) of a shared file to
	// its place in the target.
	sharedAt map[string]string
}

func placesOf(skills []*entity.TargetSkill) *places {
	out := &places{skills: map[model.SourceKey][]*entity.TargetSkill{}, sharedAt: map[string]string{}}
	for _, s := range skills {
		key := s.Origin().Repo.Key
		out.skills[key] = append(out.skills[key], s)
	}
	return out
}

// shared returns where the shared file repoPath of owner's source goes,
// copying it into owner at "files/<repoPath>" the first time it is asked
// for.
func (p *places) shared(owner *entity.TargetSkill, repoPath string) (string, error) {
	repo := owner.Origin().Repo
	key := entity.GetSkillKey(repo.Key, repoPath)
	if at, ok := p.sharedAt[key]; ok {
		return at, nil
	}
	info, err := fs.Stat(repo.FS, repoPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		panic(fmt.Sprintf("transformers: link to the folder %s outside every skill in skill %s, after validation", repoPath, owner.Name()))
	}
	content, err := fs.ReadFile(repo.FS, repoPath)
	if err != nil {
		return "", err
	}
	rel := path.Join("files", repoPath)
	f, err := owner.AddFile(rel, content)
	if err != nil {
		return "", err
	}
	f.SetMode(info.Mode())
	at := path.Join(owner.Name(), rel)
	p.sharedAt[key] = at
	return at, nil
}

// of returns where repoPath of repository key goes: the skill's marker file
// to "{name}/SKILL.md", anything else in its folder to "{name}/<path in
// the folder>"; false if no loaded skill holds repoPath.
//
// Примеры (guide в "a/b/guide", flat f в "f.skill.md"):
//   - "a/b/guide/SKILL.md" -> "guide/SKILL.md"
//   - "a/b/guide/docs"     -> "guide/docs"
//   - "a/b/guide"          -> "guide"
//   - "f.skill.md"         -> "f/SKILL.md"
//   - "a/b"                -> false
func (p *places) of(key model.SourceKey, repoPath string) (string, bool) {
	for _, s := range p.skills[key] {
		o := s.Origin()
		if repoPath == o.MainFilePath {
			return path.Join(s.Name(), "SKILL.md"), true
		}
		if o.Format == entity.FlatSkill {
			continue
		}
		switch {
		case o.SkillDirPath == "." || o.SkillDirPath == "":
			return path.Join(s.Name(), repoPath), true
		case repoPath == o.SkillDirPath:
			return s.Name(), true
		case strings.HasPrefix(repoPath, o.SkillDirPath+"/"):
			return path.Join(s.Name(), strings.TrimPrefix(repoPath, o.SkillDirPath+"/")), true
		}
	}
	return "", false
}
