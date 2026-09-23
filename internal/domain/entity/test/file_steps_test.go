package entity_test

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing/fstest"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
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
	sc.Step(`^the contract file has link target "([^"]*)"$`, func(ctx context.Context, target string) error {
		assertedFile = entity.MakeFile("docs/intro.md", usedSkill)
		return nil
	})
	sc.Step(`^the contract file link targets are "([^"]*)"$`, func(ctx context.Context, want string) error {
		panic("implement me")
		//var targets []string
		//links, err := assertedFile.Links()
		//if err != nil {
		//	return err
		//}
		//for _, link := range links {
		//	targets = append(targets, link.Target)
		//}
		//return testsupport.Equal(strings.Join(targets, ","), want)
	})
}
