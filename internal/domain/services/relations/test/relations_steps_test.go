package relations_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/relations"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
	"testing/fstest"
)

func initialize(sc *godog.ScenarioContext) {
	var enabled bool
	var tree fstest.MapFS
	var failure error
	var names []string
	sc.Step(`^relation policy "([^"]*)"$`, func(ctx context.Context, p string) error {
		enabled = p == "enabled"
		testsupport.Log("policy=%s", p)
		return nil
	})
	sc.Step(`^relation tree$`, func(ctx context.Context, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree = fstest.MapFS{}
		for p, s := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(s), Mode: 0644}
		}
		testsupport.Log("files=%v", raw)
		return nil
	})
	sc.Step(`^I expand from "([^"]*)"$`, func(ctx context.Context, start string) error {
		detector := discovery.Detector{Codec: document.Codec{}}
		repo := &model.Repository{ID: "repo", Root: "/source", FS: tree}
		skills, err := detector.Discover(ctx, repo, start)
		if err != nil {
			return err
		}
		for _, s := range skills {
			if err := discovery.LoadFiles(s); err != nil {
				return err
			}
		}
		cat := &model.Catalog{Skills: skills}
		failure = (relations.Expander{Detector: detector}).Expand(ctx, cat, enabled, []string{"examples"})
		names = []string{}
		for _, s := range cat.Skills {
			names = append(names, s.Name)
		}
		return nil
	})
	sc.Step(`^relation names are "([^"]*)" and error contains "([^"]*)"$`, func(ctx context.Context, want, contains string) error {
		if err := testsupport.Equal(strings.Join(names, ","), want); err != nil {
			return err
		}
		testsupport.Log("error=%v", failure)
		if contains == "" {
			return failure
		}
		if failure == nil || !strings.Contains(failure.Error(), contains) {
			return fmt.Errorf("error=%v want %s", failure, contains)
		}
		return nil
	})
}
