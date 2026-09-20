package links_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
	"testing/fstest"
)

func initialize(sc *godog.ScenarioContext) {
	var input string
	var actual []any
	var tree fstest.MapFS
	var target string
	var failure error
	var knownRoot string
	sc.Step(`^link document$`, func(ctx context.Context, d *godog.DocString) error {
		input = d.Content
		testsupport.Log("document=%s", input)
		return nil
	})
	sc.Step(`^I extract links from "([^"]*)"$`, func(ctx context.Context, p string) error {
		actual = []any{}
		for _, l := range links.Extract(input) {
			actual = append(actual, map[string]any{"text": l.Text, "path": l.Path, "fragment": l.Fragment, "image": l.Image, "excluded": links.Excluded(input, p, l, []string{"examples"})})
		}
		return nil
	})
	sc.Step(`^references are$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(actual, d) })
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
		target, failure = links.Resolve(&model.Repository{FS: tree, Root: "/source"}, from, raw, func(p string) bool { return knownRoot != "" && (p == knownRoot || strings.HasPrefix(p, knownRoot+"/")) })
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
