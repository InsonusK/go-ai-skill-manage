package skill_selector_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
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
}

func (p selectProvider) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	*p.calls++
	return &entity.Repository{Key: key, FS: p.tree}, nil
}

func initialize(sc *godog.ScenarioContext) {
	var tree fstest.MapFS
	var acquireCalls int
	var names []string
	var failure error
	var canceled bool

	sc.Step(`^a source tree$`, func(ctx context.Context, d *godog.DocString) error {
		var files map[string]string
		if err := json.Unmarshal([]byte(d.Content), &files); err != nil {
			return err
		}
		canceled = false
		acquireCalls = 0
		tree = fstest.MapFS{}
		for p, v := range files {
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
		testsupport.Log("files=%v", files)
		return nil
	})
	sc.Step(`^discovery is canceled$`, func(ctx context.Context) error { canceled = true; return nil })
	sc.Step(`^I select from subpaths "([^"]*)" with tags "([^"]*)"$`, func(ctx context.Context, subpaths, tagsArg string) error {
		spec := model.SourceSpec{Type: "local"}
		if subpaths != "" {
			spec.Subpaths = strings.Split(subpaths, ",")
		}
		if tagsArg != "" {
			spec.Tags = []string{tagsArg}
		}
		catalog := &sourcing.SkillCatalog{
			Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": selectProvider{tree: tree, calls: &acquireCalls}}, ""),
		}
		if canceled {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		skills, err := (skill_selector.SkillSelector{SkillCatalog: catalog}).Select(ctx, spec)
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
	sc.Step(`^the source provider was not acquired$`, func(ctx context.Context) error {
		return testsupport.Equal(acquireCalls, 0)
	})
}
