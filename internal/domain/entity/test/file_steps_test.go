package entity_test

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func registerFileSteps(sc *godog.ScenarioContext) {
	var assertedFile *entity.File
	var usedSkill *entity.Skill
	var contractCounts map[string]int
	var contractRepoPath string
	var contractContent []byte
	var contractErr error
	var requestedFilePath string
	var requestedFilePathErr error
	var contractLinks []*model.Link
	var linksErr error

	sc.Step(`^a file "([^"]*)" containing "([^"]*)" attached to a mocked skill rooted at "([^"]*)" in repository "([^"]*)"$`,
		func(ctx context.Context, filePath, content, skillRoot, repositoryRoot string) error {
			contractCounts = map[string]int{}
			contractRepoPath = path.Join(skillRoot, filePath)
			mainFilePath := path.Join(skillRoot, "SKILL.md")
			repository := &entity.Repository{
				Key:      model.SourceKey{Type: "local", Path: "repo"},
				RootPath: repositoryRoot,
				FS: countingFS{
					files: fstest.MapFS{
						contractRepoPath: &fstest.MapFile{Data: []byte(content)},
						mainFilePath:     &fstest.MapFile{Data: []byte("---\nname: guide\n---\n")},
					},
					counts: contractCounts,
				},
			}
			var err error
			usedSkill, err = entity.MakeSkill(repository, mainFilePath, skillRoot, entity.AgentDirSkill)
			if err != nil {
				return err
			}
			assertedFile = entity.MakeFile(filePath, usedSkill)
			contractContent, contractErr = nil, nil
			requestedFilePath, requestedFilePathErr = "", nil
			contractLinks, linksErr = nil, nil
			entity.SetDefaultLinkSearcher(links.Searcher{})
			return nil
		})
	sc.Step(`^I read the contract file content$`, func(ctx context.Context) error {
		contractContent, contractErr = assertedFile.Content()
		return nil
	})
	sc.Step(`^the contract file content is "([^"]*)"$`, func(ctx context.Context, want string) error {
		if contractErr != nil {
			return contractErr
		}
		return testsupport.Equal(string(contractContent), want)
	})
	sc.Step(`^the contract file was read (\d+) times?$`, func(ctx context.Context, want int) error {
		return testsupport.Equal(contractCounts[contractRepoPath], want)
	})
	sc.Step(`^I request the contract file path as "([^"]*)"$`, func(ctx context.Context, kind string) error {
		requestedFilePath, requestedFilePathErr = assertedFile.Path(entity.FilePathKind(kind))
		return nil
	})
	sc.Step(`^the requested file path is "([^"]*)"$`, func(ctx context.Context, want string) error {
		if requestedFilePathErr != nil {
			return requestedFilePathErr
		}
		return testsupport.Equal(requestedFilePath, want)
	})
	sc.Step(`^requesting the file path fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if requestedFilePathErr == nil || !strings.Contains(requestedFilePathErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", requestedFilePathErr, want)
		}
		return nil
	})
	sc.Step(`^the contract file belongs to the mocked skill$`, func(ctx context.Context) error {
		return testsupport.Equal(assertedFile.Skill().Key(), usedSkill.Key())
	})
	sc.Step(`^no link searcher is configured$`, func(ctx context.Context) error {
		entity.SetDefaultLinkSearcher(nil)
		return nil
	})
	sc.Step(`^I read the contract file links$`, func(ctx context.Context) error {
		contractLinks, linksErr = assertedFile.Links()
		return nil
	})
	sc.Step(`^the contract file link targets are "([^"]*)"$`, func(ctx context.Context, want string) error {
		if linksErr != nil {
			return linksErr
		}
		var paths []string
		for _, link := range contractLinks {
			paths = append(paths, link.Path)
		}
		return testsupport.Equal(strings.Join(paths, ","), want)
	})
	sc.Step(`^reading the contract file links fails with "([^"]*)"$`, func(ctx context.Context, want string) error {
		if linksErr == nil || !strings.Contains(linksErr.Error(), want) {
			return fmt.Errorf("error=%v want contains %s", linksErr, want)
		}
		return nil
	})
}
