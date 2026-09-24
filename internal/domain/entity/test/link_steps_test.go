package entity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// fakeSkillResolver answers TryGetOrFetchByPathUp with a preset skill or
// error and records every path it was asked about; the other lookups are
// never expected from a Link.
type fakeSkillResolver struct {
	skill *entity.Skill
	err   error
	asked []string
}

func (r *fakeSkillResolver) GetByPath(ctx context.Context, key model.SourceKey, p string) ([]*entity.Skill, error) {
	return nil, errors.New("unexpected GetByPath")
}

func (r *fakeSkillResolver) GetByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	return nil, errors.New("unexpected GetByPathUp")
}

func (r *fakeSkillResolver) TryGetOrFetchByPath(ctx context.Context, key model.SourceKey, p string) ([]*entity.Skill, error) {
	return nil, errors.New("unexpected TryGetOrFetchByPath")
}

func (r *fakeSkillResolver) TryGetOrFetchByPathUp(ctx context.Context, key model.SourceKey, p string) (*entity.Skill, error) {
	r.asked = append(r.asked, key.String()+" "+p)
	return r.skill, r.err
}

func registerLinkSteps(sc *godog.ScenarioContext) {
	var repo *entity.Repository
	var linkFile *entity.File
	var link *entity.Link
	var linkErr error
	var resolver *fakeSkillResolver
	var skillOf func(dir string) (*entity.Skill, error)

	sc.Step(`^a repository "([^"]*)" holding$`, func(ctx context.Context, repoDir string, d *godog.DocString) error {
		var raw map[string]string
		if err := json.Unmarshal([]byte(d.Content), &raw); err != nil {
			return err
		}
		tree := fstest.MapFS{}
		for p, v := range raw {
			tree[p] = &fstest.MapFile{Data: []byte(v)}
		}
		repo = &entity.Repository{Key: model.SourceKey{Type: "local", Path: "repo"}, RootPath: repoDir, FS: tree}
		skillOf = func(dir string) (*entity.Skill, error) {
			return entity.MakeSkill(repo, dir+"/SKILL.md", dir, entity.AgentDirSkill)
		}
		resolver = &fakeSkillResolver{}
		link, linkErr = nil, nil
		testsupport.Log("repository=%s files=%v", repoDir, raw)
		return nil
	})
	sc.Step(`^the file "([^"]*)" of the skill at "([^"]*)" contains$`, func(ctx context.Context, p, dir string, d *godog.DocString) error {
		repo.FS.(fstest.MapFS)[dir+"/"+p] = &fstest.MapFile{Data: []byte(d.Content)}
		skill, err := skillOf(dir)
		if err != nil {
			return err
		}
		linkFile = entity.MakeFile(p, skill)
		return nil
	})
	sc.Step(`^I read the first link of that file$`, func(ctx context.Context) error {
		entity.SetDefaultLinkSearcher(links.NewDefaultLinkFactory())
		defer entity.SetDefaultLinkSearcher(nil)
		found, err := linkFile.Links()
		if err != nil {
			return err
		}
		if len(found) == 0 {
			return errors.New("no link found")
		}
		link = found[0]
		testsupport.Log("link=%+v", *link)
		return nil
	})
	sc.Step(`^I make a link of that file from span (\d+)-(\d+)$`, func(ctx context.Context, start, end int) error {
		link, linkErr = entity.MakeLink(linkFile, model.ParsedLink{Start: start, End: end})
		return nil
	})
	sc.Step(`^making the link fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		return errorContains(linkErr, want)
	})
	sc.Step(`^the link raw is "([^"]*)" and external is (true|false)$`, func(ctx context.Context, raw, external string) error {
		return testsupport.Equal([]any{link.Raw, link.External, link.File() == linkFile}, []any{raw, external == "true", true})
	})
	sc.Step(`^the resolver finds the skill at "([^"]*)"$`, func(ctx context.Context, dir string) error {
		skill, err := skillOf(dir)
		resolver.skill = skill
		return err
	})
	sc.Step(`^the resolver has the skill not cached$`, func(ctx context.Context) error {
		resolver.err = fmt.Errorf("%w: fake", entity.ErrSkillNotCached)
		return nil
	})
	sc.Step(`^the link path as "([^"]*)" is "([^"]*)"$`, func(ctx context.Context, kind, want string) error {
		got, err := link.Path(ctx, model.PathKind(kind), resolver)
		if err != nil {
			return err
		}
		return testsupport.Equal(got, want)
	})
	sc.Step(`^the link path as "([^"]*)" fails with "([^"]*)"$`, func(ctx context.Context, kind, want string) error {
		_, err := link.Path(ctx, model.PathKind(kind), resolver)
		return errorContains(err, want)
	})
	sc.Step(`^the link path as "([^"]*)" fails as not cached$`, func(ctx context.Context, kind string) error {
		_, err := link.Path(ctx, model.PathKind(kind), resolver)
		if !errors.Is(err, entity.ErrSkillNotCached) {
			return fmt.Errorf("error=%v, want ErrSkillNotCached", err)
		}
		return nil
	})
	sc.Step(`^the link skill is the skill at "([^"]*)"$`, func(ctx context.Context, dir string) error {
		skill, err := link.Skill(ctx, resolver)
		if err != nil {
			return err
		}
		return testsupport.Equal(skill.SkillDirPath, dir)
	})
	sc.Step(`^the link skill fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		_, err := link.Skill(ctx, resolver)
		return errorContains(err, want)
	})
	sc.Step(`^the resolver was asked for "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Join(resolver.asked, ","), want)
	})
}

func errorContains(err error, want string) error {
	if err == nil || !strings.Contains(err.Error(), want) {
		return fmt.Errorf("error=%v want contains %s", err, want)
	}
	return nil
}
