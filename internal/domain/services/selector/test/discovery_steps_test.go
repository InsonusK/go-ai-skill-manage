package skill_selector_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	skill_selector "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/selector"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// selectProvider is a fake interfaces.SourceProvider returning a Repository
// backed by the scenario's tree, counting Acquire calls so a canceled-context
// scenario can prove Select never even reaches the provider.
type selectProvider struct {
	tree  fstest.MapFS
	calls *int
	// fail, when set, is the error every Acquire returns.
	fail string
}

func (p selectProvider) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	*p.calls++
	if p.fail != "" {
		return nil, errors.New(p.fail)
	}
	return &entity.Repository{Key: key, FS: p.tree}, nil
}

func initialize(sc *godog.ScenarioContext) {
	var tree fstest.MapFS
	var acquireCalls int
	var names []string
	var failure error
	var problems issues.SkillIssues
	var canceled bool
	var acquireFailure string
	var panicked string
	var catalog *sourcing.SkillCatalog

	sc.Step(`^a source tree$`, func(ctx context.Context, d *godog.DocString) error {
		var files map[string]string
		if err := json.Unmarshal([]byte(d.Content), &files); err != nil {
			return err
		}
		canceled = false
		acquireFailure = ""
		acquireCalls = 0
		tree = fstest.MapFS{}
		for p, v := range files {
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
		testsupport.Log("files=%v", files)
		return nil
	})
	sc.Step(`^discovery is canceled$`, func(ctx context.Context) error { canceled = true; return nil })
	sc.Step(`^the source provider fails with "([^"]*)"$`, func(ctx context.Context, msg string) error {
		acquireFailure = msg
		return nil
	})
	sc.Step(`^I select from subpaths "([^"]*)" with tags "([^"]*)"$`, func(ctx context.Context, subpaths, tagsArg string) error {
		spec := model.SourceSpec{Type: "local", Path: "repo"}
		if subpaths != "" {
			spec.Subpaths = strings.Split(subpaths, ",")
		}
		if tagsArg != "" {
			spec.Tags = []string{tagsArg}
		}
		catalog = &sourcing.SkillCatalog{
			Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": selectProvider{tree: tree, calls: &acquireCalls, fail: acquireFailure}}, ""),
		}
		if canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		panicked = ""
		defer func() {
			if r := recover(); r != nil {
				panicked = fmt.Sprint(r)
				testsupport.Log("panic=%s", panicked)
			}
		}()
		skills, list := (skill_selector.SkillSelector{SkillCatalog: catalog}).Select(ctx, spec)
		problems, failure = list, nil
		if len(list) > 0 {
			failure = list
		}
		names = []string{}
		for _, s := range skills {
			names = append(names, s.Name)
		}
		return nil
	})
	sc.Step(`^selection panics with "([^"]*)"$`, func(ctx context.Context, text string) error {
		if !strings.Contains(panicked, text) {
			return fmt.Errorf("panic=%q; want one with %q", panicked, text)
		}
		return nil
	})
	sc.Step(`^discovered names are "([^"]*)" and discovery error contains "([^"]*)"$`, func(ctx context.Context, want, contains string) error {
		if panicked != "" {
			return fmt.Errorf("unexpected panic: %s", panicked)
		}
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
	sc.Step(`^the catalog holds "([^"]*)"$`, func(ctx context.Context, want string) error {
		var loaded []string
		for _, s := range catalog.Skills() {
			loaded = append(loaded, s.Name)
		}
		return testsupport.Equal(strings.Join(loaded, ","), want)
	})
	sc.Step(`^the source provider was not acquired$`, func(ctx context.Context) error {
		return testsupport.Equal(acquireCalls, 0)
	})
	sc.Step(`^the source provider was acquired (\d+) times?$`, func(ctx context.Context, n int) error {
		return testsupport.Equal(acquireCalls, n)
	})
	// the selection issues are: a JSON list of [code, source, file].
	sc.Step(`^the selection issues are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			got = append(got, []string{i.Code, i.Source, i.File})
		}
		testsupport.Log("issues=%v", problems)
		return testsupport.JSON(got, d)
	})
}
