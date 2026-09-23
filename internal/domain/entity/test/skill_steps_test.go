package entity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func registerSkillSteps(sc *godog.ScenarioContext) {
	var skill *entity.Skill
	var repositoryCounts map[string]int
	var listedFiles []*entity.File
	var listErr error
	var key string
	var openCountSnapshot int
	var buildErr error

	totalOpenCount := func() int {
		total := 0
		for _, count := range repositoryCounts {
			total += count
		}
		return total
	}

	listPaths := func() ([]string, error) {
		paths := make([]string, 0, len(listedFiles))
		for _, file := range listedFiles {
			filePath, err := file.Path(entity.SkillRelative)
			if err != nil {
				return nil, err
			}
			paths = append(paths, filePath)
		}
		sort.Strings(paths)
		return paths, nil
	}

	sc.Step(`^an? "([^"]*)" skill "([^"]*)" rooted at "([^"]*)" with main file "([^"]*)" in repository "([^"]*)" and files$`,
		func(ctx context.Context, format, name, skillRoot, mainFilePath, repositoryRoot string, document *godog.DocString) error {
			var rawFiles map[string]string
			if err := json.Unmarshal([]byte(document.Content), &rawFiles); err != nil {
				return err
			}
			repositoryCounts = map[string]int{}
			tree := fstest.MapFS{}
			for filePath, content := range rawFiles {
				tree[filePath] = &fstest.MapFile{Data: []byte(content)}
			}
			repository := &entity.Repository{
				Key:      model.SourceKey{Type: "local", Path: "repo"},
				RootPath: repositoryRoot,
				FS: countingFS{
					files:  tree,
					counts: repositoryCounts,
				},
			}
			skill, buildErr = entity.MakeSkill(repository, mainFilePath, skillRoot, entity.SkillFormat(format))
			listedFiles, listErr, key = nil, nil, ""
			openCountSnapshot = 0
			testsupport.Log("created %s skill %q at %q with %d repository files", format, name, skillRoot, len(rawFiles))
			return nil
		})

	sc.Step(`^building the skill fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if buildErr == nil || !strings.Contains(buildErr.Error(), want) {
			return fmt.Errorf("error=%v; want contains %s", buildErr, want)
		}
		return nil
	})
	sc.Step(`^I request the skill key$`, func(ctx context.Context) error {
		key = skill.Key()
		testsupport.Log("requested skill key -> %q", key)
		return nil
	})
	sc.Step(`^the skill key is$`, func(ctx context.Context, expected *godog.DocString) error {
		testsupport.Log("comparing skill key %q", key)
		return testsupport.JSON(key, expected)
	})
	sc.Step(`^I request the skill metadata$`, func(ctx context.Context) error {
		testsupport.Log("requested metadata -> name=%q main=%q root=%q", skill.Name, skill.MainFilePath, skill.SkillDirPath)
		return nil
	})
	sc.Step(`^the skill metadata is$`, func(ctx context.Context, expected *godog.DocString) error {
		actual := map[string]any{
			"name":           skill.Name,
			"mainFilePath":   skill.MainFilePath,
			"skillDirPath":   skill.SkillDirPath,
			"format":         skill.Format,
			"repositoryId":   skill.Repo.Key.String(),
			"repositoryRoot": skill.Repo.RootPath,
		}
		testsupport.Log("comparing returned skill metadata")
		return testsupport.JSON(actual, expected)
	})
	sc.Step(`^I list skill files at path "([^"]*)"$`, func(ctx context.Context, requestedPath string) error {
		listedFiles, listErr = skill.FilesByPath(requestedPath)
		testsupport.Log("listed skill path %q -> %d files, error=%v", requestedPath, len(listedFiles), listErr)
		return nil
	})
	sc.Step(`^I list skill files at path "([^"]*)" matching "([^"]*)"$`, func(ctx context.Context, requestedPath, pattern string) error {
		filter, err := regexp.Compile(pattern)
		if err != nil {
			return err
		}
		listedFiles, listErr = skill.FilesByPath(requestedPath, filter)
		testsupport.Log("listed skill path %q matching %q -> %d files, error=%v", requestedPath, pattern, len(listedFiles), listErr)
		return nil
	})
	sc.Step(`^the listed skill-relative paths are$`, func(ctx context.Context, expected *godog.DocString) error {
		if listErr != nil {
			return listErr
		}
		paths, err := listPaths()
		if err != nil {
			return err
		}
		testsupport.Log("comparing listed paths -> %v", paths)
		return testsupport.JSON(paths, expected)
	})
	sc.Step(`^the repository open count is (\d+)$`, func(ctx context.Context, expected int) error {
		actual := totalOpenCount()
		testsupport.Log("comparing repository open count %d with %d", actual, expected)
		return testsupport.Equal(actual, expected)
	})
	sc.Step(`^I remember the repository open count$`, func(ctx context.Context) error {
		openCountSnapshot = totalOpenCount()
		testsupport.Log("remembered repository open count -> %d", openCountSnapshot)
		return nil
	})
	sc.Step(`^the repository open count has not increased$`, func(ctx context.Context) error {
		actual := totalOpenCount()
		testsupport.Log("comparing repository open count %d with snapshot %d", actual, openCountSnapshot)
		return testsupport.Equal(actual, openCountSnapshot)
	})
	sc.Step(`^listing skill files fails with filesystem error "([^"]*)"$`, func(ctx context.Context, kind string) error {
		if listErr == nil {
			return fmt.Errorf("expected filesystem error %q, got nil", kind)
		}
		var target error
		switch kind {
		case "not-exist":
			target = fs.ErrNotExist
		case "invalid-path":
			target = fs.ErrInvalid
		default:
			return fmt.Errorf("unknown filesystem error kind %q", kind)
		}
		testsupport.Log("checking filesystem error %v against %v", listErr, target)
		if !errors.Is(listErr, target) {
			return fmt.Errorf("error=%v; want errors.Is(_, %v)", listErr, target)
		}
		return nil
	})
}
