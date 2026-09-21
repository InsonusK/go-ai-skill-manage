package discovery

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-{1,2}[a-z0-9]+)*$`)

func ValidName(name string) bool { return namePattern.MatchString(name) }

type Detector struct{ Codec interfaces.DocumentCodec }

func (d Detector) Discover(ctx context.Context, repo *model.Repository, start string) ([]*model.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := []*model.Skill{}
	var issues model.Issues
	var scan func(string)
	scan = func(p string) {
		if err := ctx.Err(); err != nil {
			issues = append(issues, issue("canceled", p, err))
			return
		}
		info, err := fs.Stat(repo.FS, p)
		if err != nil {
			if !isNotExist(err) {
				issues = append(issues, issue("source-read", p, err))
			}
			return
		}
		if !info.IsDir() {
			if strings.HasSuffix(p, ".skill.md") {
				s, err := d.make(repo, p, "", model.FlatSkill, info.Mode().Perm(), nil)
				if err != nil {
					issues = append(issues, issue("invalid-name", p, err))
				} else {
					out = append(out, s)
				}
			}
			return
		}
		skill, err := d.Rooted(ctx, repo, p)
		if err != nil {
			issues = append(issues, issue("invalid-skill", p, err))
			return
		}
		if skill != nil {
			out = append(out, skill)
			return
		}
		entries, err := fs.ReadDir(repo.FS, p)
		if err != nil {
			issues = append(issues, issue("source-read", p, err))
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

// Rooted tests whether dir is a directory skill's root. It reads the marker
// file's own bytes (for frontmatter/name validation) but only the *paths*
// of every other nested file -- their content is loaded later, by
// LoadFiles, only for skills that survive tag/subpath filtering.
func (d Detector) Rooted(ctx context.Context, repo *model.Repository, dir string) (*model.Skill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	var mainMode fs.FileMode
	var files []model.File
	if err := fs.WalkDir(repo.FS, dir, func(p string, e fs.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			return err
		}
		if e.IsDir() && e.Name() == ".git" {
			return fs.SkipDir
		}
		if p == dir {
			return nil
		}
		if p == main {
			info, err := e.Info()
			if err != nil {
				return err
			}
			mainMode = info.Mode().Perm()
			return nil
		}
		rel := strings.TrimPrefix(p, dir+"/")
		if dir == "." {
			rel = p
		}
		first := strings.Split(rel, "/")[0]
		for _, skip := range repo.SkipFolders {
			if first == skip {
				if e.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if !e.IsDir() && (e.Name() == "SKILL.md" || strings.HasSuffix(e.Name(), ".skill.md")) {
			return fmt.Errorf("nested-skill: %s", p)
		}
		if e.IsDir() || e.Name() == model.Marker {
			return nil
		}
		info, err := e.Info()
		if err != nil {
			return err
		}
		files = append(files, model.File{Path: rel, Mode: info.Mode().Perm()})
		return nil
	}); err != nil {
		return nil, err
	}
	return d.make(repo, main, dir, format, mainMode, files)
}

// make reads the skill's own file (main), validates its frontmatter name,
// and assembles the Skill -- MainFile carries that file's bytes; files
// (already collected by Rooted, or nil for a flat skill) carries only paths
// and modes, content loaded later by LoadFiles.
func (d Detector) make(repo *model.Repository, main, root string, format model.SkillFormat, mainMode fs.FileMode, files []model.File) (*model.Skill, error) {
	data, err := fs.ReadFile(repo.FS, main)
	if err != nil {
		return nil, err
	}
	doc, err := d.Codec.Decode(data)
	if err != nil {
		return nil, err
	}
	name, _ := doc.Properties["name"].(string)
	if !ValidName(name) {
		return nil, fmt.Errorf("invalid-name: %q must use lowercase letters, digits and single/double hyphens", name)
	}
	return &model.Skill{
		Name: name, Main: main, Root: root, Format: format, Repo: repo, Document: doc,
		MainFile: model.File{Path: "SKILL.md", Data: data, Mode: mainMode},
		Files:    files,
	}, nil
}

// Find locates the skill owning a path without scanning unrelated siblings.
func (d Detector) Find(ctx context.Context, repo *model.Repository, p string) (*model.Skill, error) {
	info, err := fs.Stat(repo.FS, p)
	if err != nil {
		return nil, err
	}
	dir := p
	if !info.IsDir() {
		dir = path.Dir(p)
	}
	for {
		s, err := d.Rooted(ctx, repo, dir)
		if err != nil {
			return nil, err
		}
		if s != nil {
			return s, nil
		}
		if dir == "." {
			if strings.HasSuffix(p, ".skill.md") {
				return d.make(repo, p, "", model.FlatSkill, info.Mode().Perm(), nil)
			}
			return nil, nil
		}
		dir = path.Dir(dir)
	}
}
func issue(code, p string, err error) model.Issue {
	return model.Issue{Code: code, File: p, Message: err.Error()}
}
