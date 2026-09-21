package discovery_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
	"testing/fstest"
)

func initialize(sc *godog.ScenarioContext) {
	var tree fstest.MapFS
	var names []string
	var discovered []*model.Skill
	var failure error
	var canceled bool
	sc.Step(`^a source tree$`, func(ctx context.Context, d *godog.DocString) error {
		var files map[string]string
		if err := json.Unmarshal([]byte(d.Content), &files); err != nil {
			return err
		}
		canceled = false
		tree = fstest.MapFS{}
		for p, v := range files {
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
		testsupport.Log("files=%v", files)
		return nil
	})
	sc.Step(`^discovery is canceled$`, func(ctx context.Context) error { canceled = true; return nil })
	sc.Step(`^I find the owner of "([^"]*)"$`, func(ctx context.Context, p string) error {
		repo := &model.Repository{ID: "local", Root: "/source", FS: tree, SkipFolders: []string{"examples"}}
		skill, err := (discovery.Detector{Codec: document.Codec{}}).Find(ctx, repo, p)
		failure = err
		names = []string{}
		if skill != nil {
			names = append(names, skill.Name)
		}
		return nil
	})
	sc.Step(`^I discover skills at "([^"]*)"$`, func(ctx context.Context, p string) error {
		repo := &model.Repository{ID: "local", Root: "/source", FS: tree, SkipFolders: []string{"examples"}}
		if canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		skills, err := (discovery.Detector{Codec: document.Codec{}}).Discover(ctx, repo, p)
		failure = err
		discovered = skills
		names = []string{}
		for _, s := range skills {
			names = append(names, s.Name)
		}
		return nil
	})
	sc.Step(`^discovered skill "([^"]*)" has format "([^"]*)" root "([^"]*)" and nested files "([^"]*)"$`, func(ctx context.Context, name, format, root, nested string) error {
		var found *model.Skill
		for _, s := range discovered {
			if s.Name == name {
				found = s
				break
			}
		}
		if found == nil {
			return fmt.Errorf("skill %q not found among discovered skills", name)
		}
		if err := testsupport.Equal(string(found.Format), format); err != nil {
			return err
		}
		if err := testsupport.Equal(found.Root, root); err != nil {
			return err
		}
		if err := testsupport.Equal(found.MainFile.Path, "SKILL.md"); err != nil {
			return err
		}
		if len(found.MainFile.Data) == 0 {
			return fmt.Errorf("expected MainFile.Data to be populated")
		}
		var paths []string
		for _, f := range found.Files {
			paths = append(paths, f.Path)
			if len(f.Data) != 0 {
				return fmt.Errorf("expected nested file %q Data to stay empty until LoadFiles runs", f.Path)
			}
		}
		want := []string{}
		if nested != "" {
			want = strings.Split(nested, ",")
		}
		return testsupport.Equal(strings.Join(paths, ","), strings.Join(want, ","))
	})
	sc.Step(`^I select from subpath "([^"]*)" with tags "([^"]*)" and name "([^"]*)"$`, func(ctx context.Context, subpath, tagsArg, name string) error {
		repo := &model.Repository{ID: "local", Root: "/source", FS: tree}
		spec := model.SourceSpec{Name: name}
		if subpath != "" {
			spec.Subpaths = []string{subpath}
		}
		if tagsArg != "" {
			spec.Tags = []string{tagsArg}
		}
		skills, err := (discovery.Detector{Codec: document.Codec{}}).Select(ctx, repo, spec)
		failure = err
		names = []string{}
		for _, s := range skills {
			names = append(names, s.Name)
		}
		return nil
	})
	sc.Step(`^I select the single file "([^"]*)" with subpath "([^"]*)"$`, func(ctx context.Context, file, subpath string) error {
		repo := &model.Repository{ID: "local", Root: "/source", FS: tree, SingleFile: file}
		spec := model.SourceSpec{Subpaths: []string{subpath}}
		skills, err := (discovery.Detector{Codec: document.Codec{}}).Select(ctx, repo, spec)
		failure = err
		names = []string{}
		for _, s := range skills {
			names = append(names, s.Name)
		}
		return nil
	})
	sc.Step(`^discovered names are "([^"]*)" and discovery error contains "([^"]*)"$`, func(ctx context.Context, want, contains string) error {
		if err := testsupport.Equal(strings.Join(names, ","), want); err != nil {
			return err
		}
		testsupport.Log("error=%v", failure)
		if contains == "" {
			return failure
		}
		if failure == nil || !strings.Contains(failure.Error(), contains) {
			return fmt.Errorf("error=%v; want %q", failure, contains)
		}
		return nil
	})
}
