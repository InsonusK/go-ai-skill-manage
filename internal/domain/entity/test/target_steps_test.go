package entity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// targetSkillView is what a scenario compares of a skill: its fields and
// every file (the marker file included) as path -> content.
type targetSkillView struct {
	Name, MainFilePath, SkillDirPath, Format string
	Description                              any
	Files                                    map[string]string
}

func registerTargetSteps(sc *godog.ScenarioContext) {
	var sources []*entity.Skill
	var counts map[string]int
	var openedBefore int
	catalogs := map[string]*entity.TargetSkillCatalog{}

	opened := func() int {
		n := 0
		for _, c := range counts {
			n += c
		}
		return n
	}
	targetSkill := func(layer, name string) (*entity.TargetSkill, error) {
		c, ok := catalogs[layer]
		if !ok {
			return nil, fmt.Errorf("no %s catalog", layer)
		}
		for _, s := range c.Skills() {
			if s.Origin().Name == name {
				return s, nil
			}
		}
		return nil, fmt.Errorf("no skill %q in the %s catalog", name, layer)
	}
	targetFile := func(layer, name, p string) (*entity.TargetFile, error) {
		s, err := targetSkill(layer, name)
		if err != nil {
			return nil, err
		}
		files, err := s.Files()
		if err != nil {
			return nil, err
		}
		for _, f := range append([]*entity.TargetFile{s.MainFile()}, files...) {
			if f.Path() == p {
				return f, nil
			}
		}
		return nil, fmt.Errorf("no file %q in skill %q of the %s catalog", p, name, layer)
	}

	// source skills: markers (comma-separated) of the skills to build from
	// the tree -- "x/SKILL.md" agent-dir, "x.skill/x.skill.md" human-dir,
	// "x.skill.md" flat.
	sc.Step(`^source skills "([^"]*)" in$`, func(ctx context.Context, markers string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(v)}
		}
		counts = map[string]int{}
		repo := &entity.Repository{Key: model.SourceKey{Type: "local", Path: "repo"}, RootPath: "/repo", FS: countingFS{files: tree, counts: counts}}
		sources = nil
		clear(catalogs)
		for _, marker := range strings.Split(markers, ",") {
			dir, format := path.Dir(marker), entity.AgentDirSkill
			switch {
			case path.Base(marker) != "SKILL.md" && strings.HasSuffix(dir, ".skill"):
				format = entity.HumanDirSkill
			case path.Base(marker) != "SKILL.md":
				dir, format = "", entity.FlatSkill
			}
			s, err := entity.MakeSkill(repo, marker, dir, format)
			if err != nil {
				return err
			}
			sources = append(sources, s)
		}
		testsupport.Log("files=%v", raw)
		return nil
	})
	sc.Step(`^I remember how many files were opened$`, func(ctx context.Context) error {
		openedBefore = opened()
		return nil
	})
	sc.Step(`^no more files were opened$`, func(ctx context.Context) error {
		return testsupport.Equal(opened(), openedBefore)
	})

	sc.Step(`^I make the base target catalog$`, func(ctx context.Context) error {
		catalogs["base"] = entity.NewTargetSkillCatalog(sources)
		return nil
	})
	sc.Step(`^I clone the base target catalog as "([^"]*)"$`, func(ctx context.Context, name string) error {
		catalogs[name] = catalogs["base"].Clone()
		return nil
	})

	sc.Step(`^in the "([^"]*)" catalog I set the (name|main file path|skill dir path|format) of "([^"]*)" to "([^"]*)"$`, func(ctx context.Context, layer, field, name, value string) error {
		s, err := targetSkill(layer, name)
		if err != nil {
			return err
		}
		switch field {
		case "name":
			s.SetName(value)
		case "main file path":
			s.SetMainFilePath(value)
		case "skill dir path":
			s.SetSkillDirPath(value)
		case "format":
			s.SetFormat(entity.SkillFormat(value))
		}
		return nil
	})
	sc.Step(`^in the "([^"]*)" catalog I set the description of "([^"]*)" to "([^"]*)"$`, func(ctx context.Context, layer, name, value string) error {
		s, err := targetSkill(layer, name)
		if err != nil {
			return err
		}
		doc := s.Document()
		doc.Properties["description"] = value
		s.SetDocument(doc)
		return nil
	})
	sc.Step(`^in the "([^"]*)" catalog I change the description of "([^"]*)" to "([^"]*)" without setting the document$`, func(ctx context.Context, layer, name, value string) error {
		s, err := targetSkill(layer, name)
		if err != nil {
			return err
		}
		s.Document().Properties["description"] = value
		return nil
	})
	sc.Step(`^in the "([^"]*)" catalog I set the (path|content) of file "([^"]*)" of "([^"]*)" to "([^"]*)"$`, func(ctx context.Context, layer, field, p, name, value string) error {
		f, err := targetFile(layer, name, p)
		if err != nil {
			return err
		}
		if field == "path" {
			f.SetPath(value)
		} else {
			f.SetContent([]byte(value))
		}
		return nil
	})
	sc.Step(`^in the "([^"]*)" catalog I add file "([^"]*)" with "([^"]*)" to "([^"]*)"$`, func(ctx context.Context, layer, p, content, name string) error {
		s, err := targetSkill(layer, name)
		if err != nil {
			return err
		}
		_, err = s.AddFile(p, []byte(content))
		return err
	})

	sc.Step(`^in the "([^"]*)" catalog "([^"]*)" is$`, func(ctx context.Context, layer, name string, d *godog.DocString) error {
		s, err := targetSkill(layer, name)
		if err != nil {
			return err
		}
		files, err := s.Files()
		if err != nil {
			return err
		}
		view := targetSkillView{Name: s.Name(), MainFilePath: s.MainFilePath(), SkillDirPath: s.SkillDirPath(), Format: string(s.Format()), Description: s.Document().Properties["description"], Files: map[string]string{}}
		for _, f := range append([]*entity.TargetFile{s.MainFile()}, files...) {
			content, err := f.Content()
			if err != nil {
				return err
			}
			view.Files[f.Path()] = string(content)
		}
		return testsupport.JSON(view, d)
	})
	sc.Step(`^the source skill "([^"]*)" is$`, func(ctx context.Context, name string, d *godog.DocString) error {
		for _, s := range sources {
			if s.Name != name {
				continue
			}
			files, err := s.FilesByPath("")
			if err != nil {
				return err
			}
			view := targetSkillView{Name: s.Name, MainFilePath: s.MainFilePath, SkillDirPath: s.SkillDirPath, Format: string(s.Format), Description: s.Document.Properties["description"], Files: map[string]string{}}
			for _, f := range append([]*entity.File{s.MainFile}, files...) {
				content, err := f.Content()
				if err != nil {
					return err
				}
				rel, err := f.Path(model.SkillRelative)
				if err != nil {
					return err
				}
				view.Files[rel] = string(content)
			}
			return testsupport.JSON(view, d)
		}
		return fmt.Errorf("no source skill %q", name)
	})
	sc.Step(`^in the "([^"]*)" catalog file "([^"]*)" of "([^"]*)" has no source file$`, func(ctx context.Context, layer, p, name string) error {
		f, err := targetFile(layer, name, p)
		if err != nil {
			return err
		}
		if f.Origin() != nil {
			return fmt.Errorf("file %q has a source file", p)
		}
		return nil
	})
}
