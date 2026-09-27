package tracker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// TokenHelp says where to read how to give the tool a GitHub token.
const TokenHelp = "see docs/feedback.md"

// TokenSource finds the GitHub token the way gh does: GH_TOKEN, then
// GITHUB_TOKEN, then the token of the user logged in with gh (gh keeps it
// in the system keyring). The tool stores no token itself.
type TokenSource struct {
	Getenv func(string) string
	// GH returns the output of `gh auth token`; exec.ErrNotFound when gh
	// isn't installed.
	GH func(context.Context) (string, error)
}

func (t TokenSource) Token(ctx context.Context) (string, error) {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(t.Getenv(name)); token != "" {
			return token, nil
		}
	}
	token, err := t.GH(ctx)
	if errors.Is(err, exec.ErrNotFound) {
		return "", fmt.Errorf("no GitHub token: install gh and run gh auth login, or set GH_TOKEN (%s)", TokenHelp)
	}
	if err != nil {
		return "", fmt.Errorf("no GitHub token from gh (%w): run gh auth login, or set GH_TOKEN (%s)", err, TokenHelp)
	}
	return strings.TrimSpace(token), nil
}

// GHAuthToken runs `gh auth token`.
func GHAuthToken(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", errors.New(msg)
		}
		return "", err
	}
	return stdout.String(), nil
}
