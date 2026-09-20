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
		names = []string{}
		for _, s := range skills {
			names = append(names, s.Name)
		}
		return nil
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
