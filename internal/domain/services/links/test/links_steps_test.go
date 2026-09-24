package links_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
	"testing/fstest"
)

func initialize(sc *godog.ScenarioContext) {
	initializeFactory(sc)
	var tree fstest.MapFS
	var target string
	var failure error
	var knownRoot string
	var skipped bool
	sc.Step(`^I check whether "([^"]*)" is in skip folders "([^"]*)"$`, func(ctx context.Context, file, skip string) error {
		folders := []string{}
		if skip != "" {
			folders = strings.Split(skip, ",")
		}
		skipped = links.InSkippedFolder(file, folders)
		testsupport.Log("file=%s skip=%v skipped=%v", file, folders, skipped)
		return nil
	})
	sc.Step(`^the file is skipped: (true|false)$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(skipped, want == "true")
	})
	sc.Step(`^source paths "([^"]*)"$`, func(ctx context.Context, s string) error {
		knownRoot = ""
		tree = fstest.MapFS{}
		for _, p := range strings.Split(s, ",") {
			tree[p] = &fstest.MapFile{Data: []byte(p)}
		}
		testsupport.Log("paths=%s", s)
		return nil
	})
	sc.Step(`^known skill root "([^"]*)"$`, func(ctx context.Context, root string) error { knownRoot = root; return nil })
	sc.Step(`^I resolve "([^"]*)" from "([^"]*)"$`, func(ctx context.Context, raw, from string) error {
		target, failure = links.Resolve(&entity.Repository{FS: tree, RootPath: "/source"}, from, raw, func(p string) bool { return knownRoot != "" && (p == knownRoot || strings.HasPrefix(p, knownRoot+"/")) })
		testsupport.Log("resolved=%s error=%v", target, failure)
		return nil
	})
	sc.Step(`^the resolved path is "([^"]*)" and error contains "([^"]*)"$`, func(ctx context.Context, want, contains string) error {
		if err := testsupport.Equal(target, want); err != nil {
			return err
		}
		if contains == "" {
			return failure
		}
		if failure == nil || !strings.Contains(failure.Error(), contains) {
			return fmt.Errorf("error=%v want %s", failure, contains)
		}
		return nil
	})
}
