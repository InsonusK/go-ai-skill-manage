// Package tracker opens issues in the trackers of skill sources.
package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/infrastructure/repository"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// GitHub opens issues through the GitHub REST API
// (POST /repos/{owner}/{repo}/issues) on behalf of the token's user.
type GitHub struct {
	Client HTTPClient
	// BaseURL is the API root; empty means https://api.github.com.
	BaseURL string
	// Token gives the token to authenticate with.
	Token func(context.Context) (string, error)
}

var _ interfaces.IssueTracker = GitHub{}

// Create opens issue in the repository of source (a GitHub URL) and
// returns the issue's page. Labels the user can't set are dropped by
// GitHub, not refused.
func (g GitHub) Create(ctx context.Context, source model.SourceKey, issue model.NewIssue) (string, error) {
	owner, repo, err := repository.GitHubURL(source.Path)
	if err != nil {
		return "", fmt.Errorf("source %s: %w", source.String(), err)
	}
	token, err := g.Token(ctx)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("no GitHub token: run gh auth login, or set GH_TOKEN (%s)", TokenHelp)
	}
	payload, err := json.Marshal(issue)
	if err != nil {
		return "", err
	}
	base := g.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}
	address := base + "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/issues"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := g.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var answer struct {
		HTMLURL string `json:"html_url"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &answer)
	switch response.StatusCode {
	case http.StatusCreated:
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		// GitHub answers 404, not 403, for a repository the token can't see.
		return "", fmt.Errorf("GitHub issue in %s/%s: HTTP %d: %s: check that the token may open issues there (%s)", owner, repo, response.StatusCode, answer.Message, TokenHelp)
	default:
		return "", fmt.Errorf("GitHub issue in %s/%s: HTTP %d: %s", owner, repo, response.StatusCode, answer.Message)
	}
	if answer.HTMLURL == "" {
		return "", fmt.Errorf("GitHub issue in %s/%s: no html_url in the answer", owner, repo)
	}
	return answer.HTMLURL, nil
}
