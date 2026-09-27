package transformers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform/transformers"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// repositories is a fake interfaces.SourceProvider serving one in-memory
// tree per SourceKey.Path.
type repositories struct{ trees map[string]fstest.MapFS }

func (p repositories) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	tree, ok := p.trees[key.Path]
	if !ok {
		return nil, fmt.Errorf("no repository %q", key.Path)
	}
	return &entity.Repository{Key: key, RootPath: "/" + key.Path, FS: tree}, nil
}

func initialize(sc *godog.ScenarioContext) {
	var trees map[string]fstest.MapFS
	var catalog *sourcing.SkillCatalog
	var target *entity.TargetSkillCatalog
	var panicked string
	var logs bytes.Buffer

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		trees, target, panicked = map[string]fstest.MapFS{}, nil, ""
		logs.Reset()
		catalog = &sourcing.SkillCatalog{Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": repositories{trees: trees}}, ""), ExcludeFromChecks: map[model.SourceKey][]string{}}
		entity.SetDefaultLinkSearcher(links.NewDefaultLinkFactory())
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		entity.SetDefaultLinkSearcher(nil)
		return ctx, nil
	})

	sc.Step(`^a repository "([^"]*)" holding$`, func(ctx context.Context, name string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(v)}
		}
		trees[name] = tree
		testsupport.Log("repository=%s files=%v", name, raw)
		return nil
	})
	sc.Step(`^folders excluded from checks in "([^"]*)" are "([^"]*)"$`, func(ctx context.Context, repo, folders string) error {
		catalog.ExcludeFromChecks[model.SourceKey{Type: "local", Path: repo}] = strings.Split(folders, ",")
		return nil
	})
	sc.Step(`^skills at "([^"]*)" of "([^"]*)" are loaded$`, func(ctx context.Context, paths, repo string) error {
		for _, p := range strings.Split(paths, ",") {
			if _, err := catalog.GetOrFetchByPath(ctx, model.SourceKey{Type: "local", Path: repo}, p); err != nil {
				return err
			}
		}
		return nil
	})
	sc.Step(`^I make the target catalog$`, func(ctx context.Context) error {
		target = entity.NewTargetSkillCatalog(catalog.Skills())
		return nil
	})
	sc.Step(`^file "([^"]*)" of "([^"]*)" is already changed to "([^"]*)"$`, func(ctx context.Context, p, name, content string) error {
		for _, s := range target.Skills() {
			if s.Name() != name {
				continue
			}
			files, err := s.Files()
			if err != nil {
				return err
			}
			for _, f := range append([]*entity.TargetFile{s.MainFile()}, files...) {
				if f.Path() == p {
					f.SetContent([]byte(content))
					return nil
				}
			}
		}
		return fmt.Errorf("no file %q in %q", p, name)
	})
	sc.Step(`^I flatten the target catalog$`, func(ctx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				panicked = fmt.Sprint(r)
				testsupport.Log("panic=%s", panicked)
			}
		}()
		return transformers.FlatTransformer{Catalog: catalog}.Transform(ctx, target)
	})
	sc.Step(`^I apply the claude when-to-use transformer$`, func(ctx context.Context) error {
		previous := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
		defer slog.SetDefault(previous)
		return transformers.ClaudeWhenToUseTransformer{}.Transform(ctx, target)
	})
	// transformers: comma-separated names, run as one pipeline.
	sc.Step(`^I run the transformers "([^"]*)"$`, func(ctx context.Context, names string) error {
		known := map[string]transform.Transformer{
			"flat":               transformers.FlatTransformer{Catalog: catalog},
			"claude-when-to-use": transformers.ClaudeWhenToUseTransformer{},
			"managed-marker":     transformers.ManagedMarkerTransformer{},
		}
		var list []transform.Transformer
		for _, n := range strings.Split(names, ",") {
			list = append(list, known[n])
		}
		pipeline, err := transform.NewPipeline(list...)
		if err != nil {
			return err
		}
		return pipeline.Run(ctx, target)
	})
	sc.Step(`^the log has WARN "([^"]*)"$`, func(ctx context.Context, text string) error {
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "level=WARN") && strings.Contains(line, text) {
				return nil
			}
		}
		return fmt.Errorf("no WARN log with %q in:\n%s", text, logs.String())
	})
	sc.Step(`^the log has no WARN$`, func(ctx context.Context) error {
		if strings.Contains(logs.String(), "level=WARN") {
			return fmt.Errorf("unexpected WARN in:\n%s", logs.String())
		}
		return nil
	})
	sc.Step(`^the flattening panics with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if !strings.Contains(panicked, want) {
			return fmt.Errorf("panic=%q; want one with %q", panicked, want)
		}
		return nil
	})

	// the target skills are: [name, skill dir, marker file path, format].
	sc.Step(`^the target skills are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, s := range target.Skills() {
			got = append(got, []string{s.Name(), s.SkillDirPath(), s.MainFilePath(), string(s.Format())})
		}
		return testsupport.JSON(got, d)
	})
	// the target holds: every file as "{skill dir}/{path}" -> content.
	sc.Step(`^the target holds$`, func(ctx context.Context, d *godog.DocString) error {
		if panicked != "" {
			return fmt.Errorf("unexpected panic: %s", panicked)
		}
		got := map[string]string{}
		for _, s := range target.Skills() {
			files, err := s.Files()
			if err != nil {
				return err
			}
			for _, f := range append([]*entity.TargetFile{s.MainFile()}, files...) {
				content, err := f.Content()
				if err != nil {
					return err
				}
				key := s.SkillDirPath() + "/" + f.Path()
				if _, twice := got[key]; twice {
					return fmt.Errorf("file %s is in the target twice", key)
				}
				got[key] = string(content)
			}
		}
		return testsupport.JSON(got, d)
	})
}
