package sourcing_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// catalogTreeFS wraps fstest.MapFS, counting Open calls per path so a test
// can prove GetByPath warms a found skill's own FilesByPath("") cache
// (no re-walk on a later, separate FilesByPath call).
type catalogTreeFS struct {
	files  fstest.MapFS
	counts map[string]int
}

func (f catalogTreeFS) Open(name string) (fs.File, error) {
	f.counts[name]++
	return f.files.Open(name)
}

// catalogProvider is a fake interfaces.SourceProvider returning a
// Repository backed by catalogTreeFS, counting Acquire calls to prove
// Manager's own caching is reused across GetByPath calls.
type catalogProvider struct {
	calls *int
	fs    catalogTreeFS
}

func (p catalogProvider) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	*p.calls++
	return &entity.Repository{Key: key, FS: p.fs}, nil
}

func catalogSteps(sc *godog.ScenarioContext) {
	var catalog *sourcing.SkillCatalog
	var acquireCalls int
	var openCounts map[string]int
	var found []*entity.Skill
	var failure error
	var openSnapshot int

	newCatalog := func(tree fstest.MapFS) *sourcing.SkillCatalog {
		acquireCalls = 0
		openCounts = map[string]int{}
		provider := catalogProvider{calls: &acquireCalls, fs: catalogTreeFS{files: tree, counts: openCounts}}
		return &sourcing.SkillCatalog{
			Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": provider}, ""),
		}
	}
	sc.Step(`^a source tree$`, func(ctx context.Context, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(v), Mode: 0644}
		}
		catalog = newCatalog(tree)
		found, failure = nil, nil
		// Reset the package-level nested-skill exemption between scenarios --
		// it's a package var (SetSkipFoldersInNestedChecker), not a per-catalog
		// field, so a scenario that doesn't touch it must not inherit a
		// previous scenario's override.
		sourcing.SetSkipFoldersInNestedChecker([]string{"examples"})
		testsupport.Log("files=%v", raw)
		return nil
	})
	sc.Step(`^skip folders are "([^"]*)"$`, func(ctx context.Context, folders string) error {
		var skip []string
		if folders != "" {
			skip = strings.Split(folders, ",")
		}
		sourcing.SetSkipFoldersInNestedChecker(skip)
		return nil
	})
	sc.Step(`^I get or add skills at "([^"]*)"$`, func(ctx context.Context, p string) error {
		skills, err := catalog.GetByPath(ctx, model.SourceKey{Type: "local", Path: "repo"}, p)
		found = skills
		failure = err
		return nil
	})
	sc.Step(`^found names are "([^"]*)" and catalog error contains "([^"]*)"$`, func(ctx context.Context, want, contains string) error {
		var names []string
		for _, s := range found {
			names = append(names, s.Name)
		}
		if err := testsupport.Equal(strings.Join(names, ","), want); err != nil {
			return err
		}
		testsupport.Log("error=%v", failure)
		if contains == "" {
			return failure
		}
		if failure == nil || !strings.Contains(failure.Error(), contains) {
			return fmt.Errorf("error=%v want contains %s", failure, contains)
		}
		return nil
	})
	sc.Step(`^total directory reads so far are remembered$`, func(ctx context.Context) error {
		n := 0
		for _, c := range openCounts {
			n += c
		}
		openSnapshot = n
		return nil
	})
	sc.Step(`^I list files by path "([^"]*)" for skill "([^"]*)"$`, func(ctx context.Context, p, name string) error {
		for _, s := range found {
			if s.Name == name {
				_, err := s.FilesByPath(p)
				return err
			}
		}
		return fmt.Errorf("skill %q not found among found skills", name)
	})
	sc.Step(`^no additional directories were read$`, func(ctx context.Context) error {
		n := 0
		for _, c := range openCounts {
			n += c
		}
		return testsupport.Equal(strconv.Itoa(n), strconv.Itoa(openSnapshot))
	})
	sc.Step(`^the source manager acquired "([^"]*)" times$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strconv.Itoa(acquireCalls), want)
	})
	// SkillCatalog has no Owner/Destination yet (see AGENTS.md -- deferred
	// pending Sync integration); these steps deliberately fail so the gap
	// stays visible in `go test` output instead of being silently dropped.
	sc.Step(`^catalog owner of "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		return fmt.Errorf("sourcing.SkillCatalog has no Owner method yet -- cannot resolve owner of %q", p)
	})
	sc.Step(`^catalog owner of "([^"]*)" is not found$`, func(ctx context.Context, p string) error {
		return fmt.Errorf("sourcing.SkillCatalog has no Owner method yet -- cannot resolve owner of %q", p)
	})
	sc.Step(`^catalog destination of "([^"]*)" is "([^"]*)" at "([^"]*)"$`, func(ctx context.Context, p, name, dest string) error {
		return fmt.Errorf("sourcing.SkillCatalog has no Destination method yet -- cannot resolve destination of %q", p)
	})
}
