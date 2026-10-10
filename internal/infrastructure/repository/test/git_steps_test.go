package repository_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
	"github.com/cucumber/godog"
)

// recorder records git calls; rev-parse answers with a fixed commit.
type recorder struct{ calls [][]string }

func (r *recorder) Run(ctx context.Context, args ...string) (string, error) {
	r.calls = append(r.calls, args)
	if len(args) > 0 && args[len(args)-2] == "rev-parse" {
		return "c0ffee\n", nil
	}
	return "", nil
}
func gitSteps(sc *godog.ScenarioContext) {
	var recorded recorder
	var commit, output string
	var failure error
	sc.Step(`^I prepare a clone for branch "([^"]*)"$`, func(ctx context.Context, branch string) error {
		testsupport.Log("branch=%s", branch)
		recorded = recorder{}
		var err error
		commit, err = (repository.GitCloner{Runner: &recorded}).Clone(ctx, "https://github.com/owner/repo.git", branch, "/tmp/dest")
		return err
	})
	sc.Step(`^git calls are$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(recorded.calls, d) })
	sc.Step(`^the cloned commit is "([^"]*)"$`, func(ctx context.Context, want string) error { return testsupport.Equal(commit, want) })
	// A real repository: each row commits f.txt with content, optionally
	// tagged; commits remembers each commit's hash by its message.
	var origin, clone string
	var commits map[string]string
	git := repository.GitProcess{}
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		dir := filepath.Dir(origin)
		origin, clone = "", ""
		if dir == "." {
			return ctx, nil
		}
		return ctx, os.RemoveAll(dir)
	})
	sc.Step(`^a git repository with commits$`, func(ctx context.Context, table *godog.Table) error {
		dir, err := os.MkdirTemp("", "aism-git-test-")
		if err != nil {
			return err
		}
		origin, clone, commits = filepath.Join(dir, "origin"), filepath.Join(dir, "clone"), map[string]string{}
		if _, err := git.Run(ctx, "init", "--quiet", "--initial-branch", "master", "--", origin); err != nil {
			return err
		}
		for _, row := range table.Rows[1:] {
			message, content, tag := row.Cells[0].Value, row.Cells[1].Value, row.Cells[2].Value
			if err := os.WriteFile(filepath.Join(origin, "f.txt"), []byte(content), 0o644); err != nil {
				return err
			}
			for _, args := range [][]string{
				{"-C", origin, "add", "f.txt"},
				{"-C", origin, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "--quiet", "-m", message},
			} {
				if _, err := git.Run(ctx, args...); err != nil {
					return err
				}
			}
			hash, err := git.Run(ctx, "-C", origin, "rev-parse", "HEAD")
			if err != nil {
				return err
			}
			commits[message] = strings.TrimSpace(hash)
			if tag != "" {
				if _, err := git.Run(ctx, "-C", origin, "tag", tag); err != nil {
					return err
				}
			}
		}
		testsupport.Log("origin=%s commits=%v", origin, commits)
		return nil
	})
	// "@first" is the hash of the commit "first".
	sc.Step(`^I clone that repository at "([^"]*)"$`, func(ctx context.Context, tree string) error {
		if message, ok := strings.CutPrefix(tree, "@"); ok {
			tree = commits[message]
		}
		var err error
		commit, err = (repository.GitCloner{Runner: git}).Clone(ctx, "file://"+origin, tree, clone)
		testsupport.Log("tree=%s commit=%s error=%v", tree, commit, err)
		return err
	})
	sc.Step(`^the clone's file "([^"]*)" holds "([^"]*)"$`, func(ctx context.Context, name, want string) error {
		raw, err := os.ReadFile(filepath.Join(clone, name))
		if err != nil {
			return err
		}
		return testsupport.Equal(string(raw), want)
	})
	sc.Step(`^the clone's commit is the "([^"]*)" commit$`, func(ctx context.Context, message string) error {
		return testsupport.Equal(commit, commits[message])
	})
	sc.Step(`^I run Git with "([^"]*)"$`, func(ctx context.Context, arg string) error {
		output, failure = (repository.GitProcess{}).Run(ctx, arg)
		testsupport.Log("git=%s output=%q error=%v", arg, output, failure)
		return nil
	})
	sc.Step(`^git output starts with "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.HasPrefix(output, want), true)
	})
	sc.Step(`^I run Git in a cancelled context$`, func(ctx context.Context) error {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, failure = (repository.GitProcess{}).Run(cancelled, "version")
		testsupport.Log("error=%v", failure)
		return nil
	})
	sc.Step(`^git process error contains "([^"]*)"$`, func(ctx context.Context, w string) error {
		if w == "" {
			return failure
		}
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), w) {
			return fmt.Errorf("error=%v want %s", failure, w)
		}
		return nil
	})
}
