package validators_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/sourcing"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validators"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// fakeValidator declares a name and dependencies, returns preset issues and
// records the order validators ran in.
type fakeValidator struct {
	name   string
	deps   []validators.Dependency
	issues model.Issues
	ran    *[]string
}

func (f fakeValidator) Name() string                       { return f.name }
func (f fakeValidator) DependsOn() []validators.Dependency { return f.deps }
func (f fakeValidator) Validate(ctx context.Context, catalog *sourcing.SkillCatalog) model.Issues {
	*f.ran = append(*f.ran, f.name)
	return f.issues
}

func initialize(sc *godog.ScenarioContext) {
	var fakes map[string]*fakeValidator
	var ran []string
	var manager *validators.Manager
	var registerErr error
	var issues model.Issues

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		fakes, ran, manager, registerErr, issues = map[string]*fakeValidator{}, nil, nil, nil, nil
		return ctx, nil
	})
	sc.Step(`^fake validator "([^"]*)" depending on "([^"]*)" \((required|optional)\) reports "([^"]*)"$`, func(ctx context.Context, name, deps, kind, codes string) error {
		f := &fakeValidator{name: name, ran: &ran}
		for _, dep := range strings.Split(deps, ",") {
			if dep != "" {
				f.deps = append(f.deps, validators.Dependency{Name: dep, IsRequired: kind == "required"})
			}
		}
		for _, code := range strings.Split(codes, ",") {
			if code != "" {
				f.issues = append(f.issues, model.Issue{Code: code})
			}
		}
		fakes[name] = f
		return nil
	})
	sc.Step(`^I register validators "([^"]*)"$`, func(ctx context.Context, names string) error {
		var list []validators.Validator
		for _, name := range strings.Split(names, ",") {
			f, ok := fakes[name]
			if !ok {
				return fmt.Errorf("no fake validator %q", name)
			}
			list = append(list, *f)
		}
		manager, registerErr = validators.NewManager(list...)
		testsupport.Log("error=%v", registerErr)
		return nil
	})
	sc.Step(`^registration succeeds$`, func(ctx context.Context) error { return registerErr })
	sc.Step(`^registration fails with "(.*)"$`, func(ctx context.Context, want string) error {
		if registerErr == nil || !strings.Contains(registerErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %q", registerErr, want)
		}
		return testsupport.Equal(manager == nil, true)
	})
	sc.Step(`^I run the registered validators$`, func(ctx context.Context) error {
		if registerErr != nil {
			return registerErr
		}
		// The fakes don't read the catalog.
		issues = manager.Validate(ctx, nil)
		return nil
	})
	sc.Step(`^the validators ran in order "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(ran, ","), want)
	})
	sc.Step(`^the issue codes are "([^"]*)"$`, func(ctx context.Context, want string) error {
		var codes []string
		for _, i := range issues {
			codes = append(codes, i.Code)
		}
		return testsupport.Equal(strings.Join(codes, ","), want)
	})
}
