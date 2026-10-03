package repository

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner runs git and returns its standard output.
type Runner interface {
	Run(context.Context, ...string) (string, error)
}
type GitCloner struct{ Runner Runner }

// Clone clones tree of url into dest and returns the cloned commit. tree
// is a branch or a tag (cloned with --branch), or a full commit hash
// (IsCommit), fetched by itself: --branch takes no commit.
//
// Пример: tree "v1.2" -> git clone --depth 1 --single-branch --branch v1.2;
// tree "874b5b81d18ad24af34b2a294830f049de5c3f30" -> git init, git fetch
// --depth 1 url 874b5b..., git checkout FETCH_HEAD.
func (g GitCloner) Clone(ctx context.Context, url, tree, dest string) (string, error) {
	if tree == "" {
		tree = "master"
	}
	if IsCommit(tree) {
		if err := g.fetchCommit(ctx, url, strings.ToLower(tree), dest); err != nil {
			return "", err
		}
	} else if _, err := g.Runner.Run(ctx, "clone", "--depth", "1", "--single-branch", "--branch", tree, "--", url, dest); err != nil {
		return "", err
	}
	commit, err := g.Runner.Run(ctx, "-C", dest, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(commit), nil
}

// fetchCommit checks out commit of url into a new repository at dest.
func (g GitCloner) fetchCommit(ctx context.Context, url, commit, dest string) error {
	for _, args := range [][]string{
		{"init", "--quiet", "--", dest},
		{"-C", dest, "fetch", "--quiet", "--depth", "1", "--", url, commit},
		{"-C", dest, "checkout", "--quiet", "FETCH_HEAD"},
	} {
		if _, err := g.Runner.Run(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

// IsCommit tells whether tree is a full commit hash (SHA-1: 40 hex digits,
// SHA-256: 64). A short hash is not: git servers fetch a full one only
// (GitHub's archive takes a short one -- the fallback of Fetcher).
func IsCommit(tree string) bool {
	if len(tree) != 40 && len(tree) != 64 {
		return false
	}
	return strings.Trim(strings.ToLower(tree), "0123456789abcdef") == ""
}

type GitProcess struct{}

func (GitProcess) Run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("git: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
