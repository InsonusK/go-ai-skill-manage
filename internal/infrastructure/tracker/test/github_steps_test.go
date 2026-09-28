package tracker_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/tracker"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// received is what the fake GitHub API got.
type received struct {
	Method, Path, Accept, Authorization, APIVersion string
	Body                                            any
}

func initialize(sc *godog.ScenarioContext) {
	var server *httptest.Server
	var requests []received
	var status int
	var answer string
	var token string
	var tokenErr error
	var url string
	var failure error

	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		requests, status, answer, token, tokenErr, url, failure = []received{}, http.StatusCreated, "{}", "secret", nil, "", nil
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			var body any
			_ = json.Unmarshal(raw, &body)
			requests = append(requests, received{r.Method, r.URL.EscapedPath(), r.Header.Get("Accept"), r.Header.Get("Authorization"), r.Header.Get("X-GitHub-Api-Version"), body})
			w.WriteHeader(status)
			_, _ = w.Write([]byte(answer))
		}))
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		server.Close()
		return ctx, nil
	})

	sc.Step(`^GitHub answers (\d+) with$`, func(ctx context.Context, code string, d *godog.DocString) (err error) {
		status, err = strconv.Atoi(code)
		answer = d.Content
		return err
	})
	sc.Step(`^the token is "([^"]*)"$`, func(ctx context.Context, t string) error { token = t; return nil })
	sc.Step(`^the token can't be read: "([^"]*)"$`, func(ctx context.Context, msg string) error { tokenErr = errors.New(msg); return nil })
	sc.Step(`^I open an issue titled "([^"]*)" with body "([^"]*)" and labels "([^"]*)" in "([^"]*)"$`, func(ctx context.Context, title, body, labels, source string) error {
		g := tracker.GitHub{Client: server.Client(), BaseURL: server.URL, Token: func(context.Context) (string, error) { return token, tokenErr }}
		url, failure = g.Create(ctx, model.SourceKey{Type: "github", Path: source, Tree: "main"}, model.NewIssue{Title: title, Body: body, Labels: strings.Split(labels, ",")})
		testsupport.Log("url=%s error=%v", url, failure)
		return nil
	})
	sc.Step(`^the issue is "([^"]*)"$`, func(ctx context.Context, want string) error {
		if failure != nil {
			return failure
		}
		return testsupport.Equal(url, want)
	})
	sc.Step(`^opening fails with "(.*)"$`, func(ctx context.Context, want string) error {
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %q", failure, want)
		}
		return nil
	})
	var env map[string]string
	var gh string
	var ghErr error
	var found string
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		env, gh, ghErr, found = map[string]string{}, "", nil, ""
		return ctx, nil
	})
	sc.Step(`^the environment has ([A-Z_]+)="([^"]*)"$`, func(ctx context.Context, name, value string) error {
		env[name] = value
		return nil
	})
	sc.Step(`^gh gives the token "([^"]*)"$`, func(ctx context.Context, t string) error { gh = t + "\n"; return nil })
	sc.Step(`^gh isn't installed$`, func(ctx context.Context) error { ghErr = fmt.Errorf("gh: %w", exec.ErrNotFound); return nil })
	sc.Step(`^gh fails with "([^"]*)"$`, func(ctx context.Context, msg string) error { ghErr = errors.New(msg); return nil })
	sc.Step(`^I look for the GitHub token$`, func(ctx context.Context) error {
		source := tracker.TokenSource{Getenv: func(name string) string { return env[name] }, GH: func(context.Context) (string, error) { return gh, ghErr }}
		found, failure = source.Token(ctx)
		testsupport.Log("token=%q error=%v", found, failure)
		return nil
	})
	sc.Step(`^the token found is "([^"]*)"$`, func(ctx context.Context, want string) error {
		if failure != nil {
			return failure
		}
		return testsupport.Equal(found, want)
	})
	sc.Step(`^looking for the token fails with "(.*)"$`, func(ctx context.Context, want string) error {
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %q", failure, want)
		}
		return nil
	})
	sc.Step(`^GitHub received$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(requests, d) })
}
