package transformers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform/transformers"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// repositories is a fake interfaces.SourceProvider serving one in-memory
// tree per SourceKey.Path.
type repositories struct {
	trees   map[string]fstest.MapFS
	commits map[string]string
}

func (p repositories) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	tree, ok := p.trees[key.Path]
	if !ok {
		return nil, fmt.Errorf("no repository %q", key.Path)
	}
	return &entity.Repository{Key: key, RootPath: "/" + key.Path, Commit: p.commits[key.Path], FS: tree}, nil
}

func initialize(sc *godog.ScenarioContext) {
	var trees map[string]fstest.MapFS
	var commits map[string]string
	var catalog *sourcing.SkillCatalog
	var target *entity.TargetSkillCatalog
	var panicked string
	var warnings issues.SkillIssues

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		trees, commits, target, panicked = map[string]fstest.MapFS{}, map[string]string{}, nil, ""
		warnings = nil
		catalog = &sourcing.SkillCatalog{Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": repositories{trees: trees, commits: commits}}, ""), ExcludeFromChecks: map[model.SourceKey][]string{}}
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
	sc.Step(`^repository "([^"]*)" is at commit "([^"]*)"$`, func(ctx context.Context, name, commit string) error {
		commits[name] = commit
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
		_, err = transformers.FlatTransformer{Catalog: catalog}.Transform(ctx, target)
		return err
	})
	sc.Step(`^I apply the claude when-to-use transformer$`, func(ctx context.Context) error {
		var err error
		warnings, err = transformers.ClaudeWhenToUseTransformer{}.Transform(ctx, target)
		testsupport.Log("warnings=%v", warnings)
		return err
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
		_, err = pipeline.Run(ctx, target)
		return err
	})
	// warnings: [[code, source, skill, skill path, file], ...].
	sc.Step(`^the transformer warnings are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, w := range warnings {
			got = append(got, []string{string(w.Code), w.Source, w.Skill, w.SkillPath, w.File})
		}
		return testsupport.JSON(got, d)
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
