package model_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var sources *model.SourceMap
	sc.Step(`^a source map$`, func(ctx context.Context) error {
		sources = model.NewSourceMap()
		return nil
	})
	sc.Step(`^I put repository "([^"]*)" and repository "([^"]*)"$`, func(ctx context.Context, a, b string) error {
		sources.Put(&model.Repository{ID: a, Root: "/root-" + a})
		sources.Put(&model.Repository{ID: b, Root: "/root-" + b})
		return nil
	})
	sc.Step(`^I put repository "([^"]*)" again with root "([^"]*)"$`, func(ctx context.Context, id, root string) error {
		sources.Put(&model.Repository{ID: id, Root: root})
		return nil
	})
	sc.Step(`^repositories in order are "([^"]*)"$`, func(ctx context.Context, want string) error {
		var ids []string
		for _, r := range sources.Repositories() {
			ids = append(ids, r.ID)
		}
		return testsupport.Equal(strings.Join(ids, ","), want)
	})
	sc.Step(`^repository "([^"]*)" root is "([^"]*)"$`, func(ctx context.Context, id, want string) error {
		repo := sources.Get(id)
		if repo == nil {
			return fmt.Errorf("repository %s not found", id)
		}
		return testsupport.Equal(repo.Root, want)
	})
	sc.Step(`^repository "([^"]*)" is not found$`, func(ctx context.Context, id string) error {
		if repo := sources.Get(id); repo != nil {
			return fmt.Errorf("expected %s not found, got root %s", id, repo.Root)
		}
		return nil
	})
}
