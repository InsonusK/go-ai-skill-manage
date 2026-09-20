package repository

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(context.Context, ...string) error
}
type GitCloner struct{ Runner Runner }

func (g GitCloner) Clone(ctx context.Context, url, tree, dest string) error {
	if tree == "" {
		tree = "master"
	}
	return g.Runner.Run(ctx, "clone", "--depth", "1", "--single-branch", "--branch", tree, "--", url, dest)
}

type GitProcess struct{}

func (GitProcess) Run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("git: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
