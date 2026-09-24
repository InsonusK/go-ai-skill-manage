package validators_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator/validators"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// sourcesProvider is a fake interfaces.SourceProvider serving one in-memory
// tree per SourceKey.Path.
type sourcesProvider struct{ trees map[string]fstest.MapFS }

func (p sourcesProvider) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	tree, ok := p.trees[key.Path]
	if !ok {
		return nil, fmt.Errorf("no source %q", key.Path)
	}
	return &entity.Repository{Key: key, RootPath: "/" + key.Path, FS: tree}, nil
}

func initialize(sc *godog.ScenarioContext) {
	var trees map[string]fstest.MapFS
	var catalog *sourcing.SkillCatalog
	var issues model.Issues
	var registerErr error

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		trees = map[string]fstest.MapFS{}
		catalog = &sourcing.SkillCatalog{Manager: sourcing.NewManager(map[string]interfaces.SourceProvider{"local": sourcesProvider{trees: trees}}, "")}
		issues, registerErr = nil, nil
		entity.SetDefaultLinkSearcher(links.NewDefaultLinkFactory())
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		entity.SetDefaultLinkSearcher(nil)
		return ctx, nil
	})

	// --- sources and catalog
	sc.Step(`^a source "([^"]*)" holding$`, func(ctx context.Context, name string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(v)}
		}
		trees[name] = tree
		testsupport.Log("source=%s files=%v", name, raw)
		return nil
	})
	sc.Step(`^add relations is "(true|false)"$`, func(ctx context.Context, v string) error {
		catalog.AddRelations = v == "true"
		return nil
	})
	sc.Step(`^skills at "([^"]*)" of source "([^"]*)" are selected$`, func(ctx context.Context, paths, source string) error {
		for _, p := range strings.Split(paths, ",") {
			if _, err := catalog.GetOrFetchByPath(ctx, model.SourceKey{Type: "local", Path: source}, p); err != nil {
				return err
			}
		}
		return nil
	})
	sc.Step(`^the loaded skills are "([^"]*)"$`, func(ctx context.Context, want string) error {
		var names []string
		for _, s := range catalog.Skills() {
			names = append(names, s.Name)
		}
		return testsupport.Equal(strings.Join(names, ","), want)
	})

	// --- running validators
	sc.Step(`^I validate links$`, func(ctx context.Context) error {
		issues = validators.LinkValidator{SkipFolders: []string{"examples"}}.Validate(ctx, catalog)
		testsupport.Log("issues=%v", issues)
		return nil
	})
	sc.Step(`^I validate skill names$`, func(ctx context.Context) error {
		issues = validators.SkillNameValidator{}.Validate(ctx, catalog)
		testsupport.Log("issues=%v", issues)
		return nil
	})
	sc.Step(`^I validate links and then skill names$`, func(ctx context.Context) error {
		m, err := validator.NewManager(validators.LinkValidator{SkipFolders: []string{"examples"}}, validators.SkillNameValidator{})
		if err != nil {
			return err
		}
		issues = m.Validate(ctx, catalog)
		testsupport.Log("issues=%v", issues)
		return nil
	})
	sc.Step(`^I register validators "([^"]*)"$`, func(ctx context.Context, names string) error {
		var list []validator.Validator
		for _, name := range strings.Split(names, ",") {
			switch name {
			case "link-validator":
				list = append(list, validators.LinkValidator{})
			case "skill-name-validator":
				list = append(list, validators.SkillNameValidator{})
			default:
				return fmt.Errorf("unknown validator %q", name)
			}
		}
		_, registerErr = validator.NewManager(list...)
		testsupport.Log("error=%v", registerErr)
		return nil
	})
	sc.Step(`^registration succeeds$`, func(ctx context.Context) error { return registerErr })
	sc.Step(`^registration fails with "(.*)"$`, func(ctx context.Context, want string) error {
		if registerErr == nil || !strings.Contains(registerErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %q", registerErr, want)
		}
		return nil
	})

	// --- issues
	sc.Step(`^there are no issues$`, func(ctx context.Context) error {
		return testsupport.Equal(len(issues), 0)
	})
	// Each issue as [Code, Source, Skill, SkillPath, File, Link]; Message is
	// checked separately where it matters.
	sc.Step(`^the issues are$`, func(ctx context.Context, d *godog.DocString) error {
		actual := [][]string{}
		for _, i := range issues {
			actual = append(actual, []string{i.Code, i.Source, i.Skill, i.SkillPath, i.File, i.Link})
		}
		return testsupport.JSON(actual, d)
	})
	sc.Step(`^the issue codes are "([^"]*)"$`, func(ctx context.Context, want string) error {
		var codes []string
		for _, i := range issues {
			codes = append(codes, i.Code)
		}
		return testsupport.Equal(strings.Join(codes, ","), want)
	})
	sc.Step(`^issue (\d+) message contains "([^"]*)"$`, func(ctx context.Context, n int, want string) error {
		if n > len(issues) {
			return fmt.Errorf("only %d issues", len(issues))
		}
		if msg := issues[n-1].Message; !strings.Contains(msg, want) {
			return fmt.Errorf("message %q does not contain %q", msg, want)
		}
		return nil
	})
}
