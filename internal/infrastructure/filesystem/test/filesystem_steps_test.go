package filesystem_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/filesystem"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func initialize(sc *godog.ScenarioContext) {
	var dir string
	var skill *entity.TargetSkill
	var failure error
	var state map[string]model.Managed

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		dir, skill, failure, state = "", nil, nil, nil
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if dir != "" {
			return ctx, os.RemoveAll(dir)
		}
		return ctx, nil
	})
	sc.Step(`^an empty target$`, func(ctx context.Context) error {
		var err error
		dir, err = os.MkdirTemp("", "aism-test-")
		testsupport.Log("target=%s", dir)
		return err
	})
	sc.Step(`^target fixture file "([^"]*)" contains "([^"]*)"$`, func(ctx context.Context, p, data string) error {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, p), []byte(data), 0o644)
	})
	sc.Step(`^target symlink "([^"]*)" points to "([^"]*)"$`, func(ctx context.Context, p, dest string) error {
		return os.Symlink(dest, filepath.Join(dir, p))
	})
	sc.Step(`^unmanaged target directory "([^"]*)"$`, func(ctx context.Context, p string) error {
		return os.MkdirAll(filepath.Join(dir, p), 0o755)
	})

	// the skill to write: its files by path from its folder; SKILL.md holds
	// its name. The marker file is added unless the skill is "unmarked".
	sc.Step(`^a(n unmarked)? skill to write with files$`, func(ctx context.Context, noMarker string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			tree["src/"+p] = &fstest.MapFile{Data: []byte(v), Mode: 0o644}
		}
		repo := &entity.Repository{Key: model.SourceKey{Type: "local", Path: "repo"}, FS: tree}
		source, err := entity.MakeSkill(repo, "src/SKILL.md", "src", entity.AgentDirSkill)
		if err != nil {
			return err
		}
		skill = entity.NewTargetSkillCatalog([]*entity.Skill{source}).Skills()[0]
		if noMarker == "" {
			_, err = skill.AddFile(model.Marker, []byte("{}\n"))
		}
		return err
	})
	sc.Step(`^file "([^"]*)" of the skill to write has mode "([0-7]+)"$`, func(ctx context.Context, p, mode string) error {
		m, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			return err
		}
		files, err := skill.Files()
		if err != nil {
			return err
		}
		for _, f := range files {
			if f.Path() == p {
				f.SetMode(fs.FileMode(m))
				return nil
			}
		}
		return fmt.Errorf("no file %q", p)
	})
	sc.Step(`^I apply "(create|update|remove)" for "([^"]*)"$`, func(ctx context.Context, action, name string) error {
		op := entity.TargetOperation{Action: entity.TargetAction(action), Name: name}
		if action != "remove" {
			op.Skill = skill
		}
		failure = (filesystem.Store{}).Apply(ctx, entity.TargetPlan{Target: model.Target{Path: dir}, Operations: []entity.TargetOperation{op}})
		testsupport.Log("apply=%s %s error=%v", action, name, failure)
		return nil
	})
	sc.Step(`^I apply "create" for "([^"]*)" into a target reached through a symlink$`, func(ctx context.Context, name string) error {
		link := dir + "-link"
		if err := os.Symlink(dir, link); err != nil {
			return err
		}
		defer os.Remove(link)
		failure = (filesystem.Store{}).Apply(ctx, entity.TargetPlan{Target: model.Target{Path: filepath.Join(link, "skills")}, Operations: []entity.TargetOperation{{Action: entity.CreateTarget, Name: name, Skill: skill}}})
		return nil
	})

	sc.Step(`^applying succeeds$`, func(ctx context.Context) error { return failure })
	// the target holds: what is on disk, whatever Apply returned.
	sc.Step(`^the target holds$`, func(ctx context.Context, d *godog.DocString) error {
		got := map[string]string{}
		err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(p)
			got[filepath.ToSlash(rel)] = string(raw)
			return err
		})
		if err != nil {
			return err
		}
		return testsupport.JSON(got, d)
	})
	sc.Step(`^target file "([^"]*)" has mode "([0-7]+)"$`, func(ctx context.Context, p, want string) error {
		info, err := os.Stat(filepath.Join(dir, p))
		if err != nil {
			return err
		}
		return testsupport.Equal(fmt.Sprintf("%o", info.Mode().Perm()), want)
	})
	sc.Step(`^target path "([^"]*)" exists "(true|false)"$`, func(ctx context.Context, p, want string) error {
		_, err := os.Lstat(filepath.Join(dir, p))
		return testsupport.Equal(err == nil, want == "true")
	})
	sc.Step(`^applying fails with "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), s) {
			return fmt.Errorf("error=%v want %s", failure, s)
		}
		return nil
	})

	sc.Step(`^I take a snapshot of the target( folder that doesn't exist)?$`, func(ctx context.Context, missing string) error {
		target := dir
		if missing != "" {
			target = filepath.Join(dir, "missing")
		}
		var err error
		state, err = (filesystem.Store{}).Snapshot(ctx, target)
		return err
	})
	// the snapshot is: a JSON map of entry name -> [exists, managed].
	sc.Step(`^the snapshot is$`, func(ctx context.Context, d *godog.DocString) error {
		got := map[string][]bool{}
		for name, m := range state {
			got[name] = []bool{m.Exists, m.Managed}
		}
		return testsupport.JSON(got, d)
	})
}
