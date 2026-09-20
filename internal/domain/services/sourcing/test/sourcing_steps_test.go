package sourcing_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strconv"
	"strings"
)

// countingProvider is a fake interfaces.SourceProvider recording how many
// times it was asked to acquire, and the order its issued repositories were
// closed in -- exactly what Manager needs a real provider for is irrelevant
// to proving dedup/dispatch/Close ordering.
type countingProvider struct {
	calls  *int
	closed *[]string
}

func (p countingProvider) Acquire(ctx context.Context, s model.SourceSpec, options model.AcquisitionOptions) (*model.Repository, error) {
	*p.calls++
	repo := &model.Repository{ID: s.Path}
	id := s.Path
	repo.AddCloser(func() error { *p.closed = append(*p.closed, id); return nil })
	return repo, nil
}

func initialize(sc *godog.ScenarioContext) {
	var manager *sourcing.Manager
	var calls int
	var closedOrder []string
	var repos []*model.Repository
	var failure error
	sc.Step(`^a source manager with a counting local provider$`, func(ctx context.Context) error {
		calls = 0
		closedOrder = nil
		repos = nil
		failure = nil
		manager = sourcing.NewManager(map[string]interfaces.SourceProvider{
			"local": countingProvider{calls: &calls, closed: &closedOrder},
		})
		return nil
	})
	sc.Step(`^I acquire "([^"]*)" source "([^"]*)" twice$`, func(ctx context.Context, typ, path string) error {
		a, err := manager.Acquire(ctx, model.SourceSpec{Type: typ, Path: path}, model.AcquisitionOptions{})
		if err != nil {
			failure = err
			return nil
		}
		b, err := manager.Acquire(ctx, model.SourceSpec{Type: typ, Path: path}, model.AcquisitionOptions{})
		failure = err
		repos = []*model.Repository{a, b}
		return nil
	})
	sc.Step(`^I acquire "([^"]*)" source "([^"]*)" and "([^"]*)" source "([^"]*)"$`, func(ctx context.Context, t1, p1, t2, p2 string) error {
		_, err := manager.Acquire(ctx, model.SourceSpec{Type: t1, Path: p1}, model.AcquisitionOptions{})
		if err != nil {
			failure = err
			return nil
		}
		_, err = manager.Acquire(ctx, model.SourceSpec{Type: t2, Path: p2}, model.AcquisitionOptions{})
		failure = err
		return nil
	})
	sc.Step(`^I acquire "([^"]*)" source "([^"]*)"$`, func(ctx context.Context, typ, path string) error {
		_, failure = manager.Acquire(ctx, model.SourceSpec{Type: typ, Path: path}, model.AcquisitionOptions{})
		return nil
	})
	sc.Step(`^the provider was called "([^"]*)" times$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strconv.Itoa(calls), want)
	})
	sc.Step(`^both acquisitions returned the same repository$`, func(ctx context.Context) error {
		if repos[0] != repos[1] {
			return fmt.Errorf("expected the same repository pointer, got %p and %p", repos[0], repos[1])
		}
		return nil
	})
	sc.Step(`^acquiring fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %s", failure, want)
		}
		return nil
	})
	sc.Step(`^I close the source manager$`, func(ctx context.Context) error { return manager.Close(ctx) })
	sc.Step(`^repositories were closed in order "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(closedOrder, ","), want)
	})
}
