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

// Clone clones tree of url into dest and returns the cloned commit.
func (g GitCloner) Clone(ctx context.Context, url, tree, dest string) (string, error) {
	if tree == "" {
		tree = "master"
	}
	if _, err := g.Runner.Run(ctx, "clone", "--depth", "1", "--single-branch", "--branch", tree, "--", url, dest); err != nil {
		return "", err
	}
	commit, err := g.Runner.Run(ctx, "-C", dest, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(commit), nil
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
