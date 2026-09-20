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
				s, err := d.make(repo, p, true)
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
	if err := fs.WalkDir(repo.FS, dir, func(p string, e fs.DirEntry, err error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err != nil {
			return err
		}
		if p == dir || p == main {
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
		return nil
	}); err != nil {
		return nil, err
	}
	return d.make(repo, main, false)
}
func (d Detector) make(repo *model.Repository, main string, flat bool) (*model.Skill, error) {
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
	root := path.Dir(main)
	if flat {
		root = main
	}
	return &model.Skill{Name: name, Main: main, Root: root, Flat: flat, Repo: repo, Document: doc}, nil
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
				return d.make(repo, p, true)
			}
			return nil, nil
		}
		dir = path.Dir(dir)
	}
}
func issue(code, p string, err error) model.Issue {
	return model.Issue{Code: code, File: p, Message: err.Error()}
}
