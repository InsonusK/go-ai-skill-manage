package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/handler"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// repositories is a fake interfaces.SourceProvider serving one in-memory
// tree per SourceKey.Path; a path it has no tree for can't be acquired.
type repositories struct{ trees map[string]fstest.MapFS }

func (p repositories) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	tree, ok := p.trees[key.Path]
	if !ok {
		return nil, fmt.Errorf("no repository %q", key.Path)
	}
	return &entity.Repository{Key: key, RootPath: "/" + key.Path, FS: tree}, nil
}

func list(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func initialize(sc *godog.ScenarioContext) {
	var trees map[string]fstest.MapFS
	var req model.Request
	var catalog *sourcing.SkillCatalog
	var problems issues.SkillIssues

	registerSyncSteps(sc, &trees, &req, &problems)

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		trees, req, catalog, problems = map[string]fstest.MapFS{}, model.Request{}, nil, nil
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
	// the sources are: a table of repository, subpath, tags and
	// exclude_from_checks, lists comma-separated.
	sc.Step(`^the sources are$`, func(ctx context.Context, t *godog.Table) error {
		for _, row := range t.Rows[1:] {
			c := row.Cells
			req.Sources = append(req.Sources, model.SourceSpec{Type: "local", Path: c[0].Value, Subpaths: list(c[1].Value), Tags: list(c[2].Value), ExcludeFromChecks: list(c[3].Value)})
		}
		return nil
	})
	sc.Step(`^add relations is "(true|false)"$`, func(ctx context.Context, v string) error {
		req.AddRelations = v == "true"
		return nil
	})
	sc.Step(`^folders excluded from checks everywhere are "([^"]*)"$`, func(ctx context.Context, v string) error {
		req.ExcludeFromChecks = list(v)
		return nil
	})

	sc.Step(`^I fetch and validate skills$`, func(ctx context.Context) error {
		manager := sourcing.NewManager(map[string]interfaces.SourceProvider{"local": repositories{trees: trees}}, "")
		catalog, problems = handler.FetchAndValidateSkills(ctx, manager, req)
		testsupport.Log("issues=%v", problems)
		return nil
	})

	sc.Step(`^the catalog holds "([^"]*)"$`, func(ctx context.Context, want string) error {
		var names []string
		for _, s := range catalog.Skills() {
			names = append(names, s.Repo.Key.Path+":"+s.Name)
		}
		return testsupport.Equal(strings.Join(names, ","), want)
	})
	sc.Step(`^there are no issues$`, func(ctx context.Context) error {
		if len(problems) > 0 {
			return fmt.Errorf("unexpected issues:\n%v", problems)
		}
		return nil
	})
	// the issues are: a JSON list of [Code, Source, Skill, File, Link].
	sc.Step(`^the issues are$`, func(ctx context.Context, d *godog.DocString) error {
		got := [][]string{}
		for _, i := range problems {
			got = append(got, []string{i.Code, i.Source, i.Skill, i.File, i.Link})
		}
		return testsupport.JSON(got, d)
	})
}
