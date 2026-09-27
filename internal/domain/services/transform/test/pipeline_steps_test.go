package transform_test

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// fakeTransformer records that it ran and fails when told to.
type fakeTransformer struct {
	name string
	fail bool
	ran  *[]string
}

func (f fakeTransformer) Name() string { return f.name }
func (f fakeTransformer) Transform(ctx context.Context, catalog *entity.TargetSkillCatalog) error {
	*f.ran = append(*f.ran, f.name)
	if f.fail {
		return errors.New("broken")
	}
	return nil
}

func initialize(sc *godog.ScenarioContext) {
	var ran []string
	var list []transform.Transformer
	var pipeline *transform.Pipeline
	var catalog *entity.TargetSkillCatalog
	var failure error
	var canceled bool

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		ran, list, pipeline, failure, canceled = nil, nil, nil, nil, false
		catalog = entity.NewTargetSkillCatalog(nil)
		return ctx, nil
	})
	// transformers: comma-separated names; "name!" fails.
	sc.Step(`^transformers "([^"]*)"$`, func(ctx context.Context, names string) error {
		for _, n := range strings.Split(names, ",") {
			list = append(list, fakeTransformer{name: strings.TrimSuffix(n, "!"), fail: strings.HasSuffix(n, "!"), ran: &ran})
		}
		return nil
	})
	sc.Step(`^the catalog was already transformed by "([^"]*)"$`, func(ctx context.Context, name string) error {
		catalog.AddApplied(name)
		return nil
	})
	sc.Step(`^the run is canceled$`, func(ctx context.Context) error { canceled = true; return nil })
	sc.Step(`^I make a pipeline of them$`, func(ctx context.Context) error {
		pipeline, failure = transform.NewPipeline(list...)
		return nil
	})
	sc.Step(`^I run the pipeline$`, func(ctx context.Context) error {
		if failure != nil {
			return failure
		}
		if canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		failure = pipeline.Run(ctx, catalog)
		testsupport.Log("error=%v", failure)
		return nil
	})
	sc.Step(`^the transformers ran in order "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(ran, ","), want)
	})
	sc.Step(`^the catalog has applied "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(catalog.Applied(), ","), want)
	})
	sc.Step(`^the pipeline succeeds$`, func(ctx context.Context) error { return failure })
	sc.Step(`^the pipeline fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v; want one with %q", failure, want)
		}
		return nil
	})
}
