package repository_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/repository"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cloneStub struct{ fail bool }

func (s cloneStub) Clone(ctx context.Context, url, tree, dest string) error {
	if s.fail {
		return fmt.Errorf("clone unavailable")
	}
	return fixture(dest)
}

type archiveStub struct{ calls *int }

func (s archiveStub) Fetch(ctx context.Context, url, tree, dest string) (string, error) {
	*s.calls++
	return dest, fixture(dest)
}
func fixture(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "skills/a.skill.md"), []byte("content"), 0644)
}
func initialize(sc *godog.ScenarioContext) {
	gitSteps(sc)
	managerSteps(sc)
	var temp string
	var repo *model.Repository
	var failure error
	var calls int
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if repo != nil {
			if e := repo.Close(); e != nil {
				return ctx, e
			}
		}
		return ctx, os.RemoveAll(temp)
	})
	sc.Step(`^a repository source$`, func(ctx context.Context) error {
		var err error
		temp, err = os.MkdirTemp("", "aism-repo-test-")
		calls = 0
		repo = nil
		failure = nil
		testsupport.Log("source=%s", temp)
		if err != nil {
			return err
		}
		return fixture(temp)
	})
	sc.Step(`^I acquire the local source$`, func(ctx context.Context) error {
		var err error
		repo, err = (repository.Local{}).Acquire(ctx, model.SourceSpec{Type: "local", Path: temp}, model.AcquisitionOptions{})
		return err
	})
	sc.Step(`^I acquire the local source at "([^"]*)"$`, func(ctx context.Context, relative string) error {
		var err error
		repo, err = (repository.Local{}).Acquire(ctx, model.SourceSpec{Type: "local", Path: filepath.Join(temp, relative)}, model.AcquisitionOptions{})
		return err
	})
	sc.Step(`^I fetch GitHub with clone failure "([^"]*)"$`, func(ctx context.Context, s string) error {
		fetcher := repository.Fetcher{Git: cloneStub{fail: s == "true"}, Archive: archiveStub{calls: &calls}}
		var err error
		repo, err = fetcher.Acquire(ctx, model.SourceSpec{Type: "github", Path: "https://github.com/owner/repo.git", Tree: "main"}, model.AcquisitionOptions{TempDir: temp})
		return err
	})
	sc.Step(`^I fetch GitHub in the configured temporary directory$`, func(ctx context.Context) error {
		fetcher := repository.Fetcher{Git: cloneStub{}, Archive: archiveStub{calls: &calls}}
		testsupport.Log("configured temp dir=%s", temp)
		var err error
		repo, err = fetcher.Acquire(ctx, model.SourceSpec{Type: "github", Path: "https://github.com/owner/repo.git", Tree: "main"}, model.AcquisitionOptions{TempDir: temp})
		return err
	})
	sc.Step(`^I extract an archive with path "([^"]*)"$`, func(ctx context.Context, p string) error {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		tw := tar.NewWriter(gz)
		body := "content"
		if err := tw.WriteHeader(&tar.Header{Name: p, Mode: 0644, Size: int64(len(body))}); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			return err
		}
		if err := tw.Close(); err != nil {
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(b.Bytes()) }))
		defer server.Close()
		fetcher := repository.Archive{Client: server.Client(), BaseURL: server.URL}
		root, err := fetcher.Fetch(ctx, "https://github.com/owner/repo", "main", filepath.Join(temp, "archive"))
		failure = err
		if err == nil {
			repo, err = (repository.Local{}).Acquire(ctx, model.SourceSpec{Path: root}, model.AcquisitionOptions{})
			return err
		}
		return nil
	})
	sc.Step(`^acquired repository single file equals "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(repo.SingleFile, want)
	})
	sc.Step(`^acquired file "([^"]*)" equals "([^"]*)"$`, func(ctx context.Context, p, want string) error {
		raw, err := fs.ReadFile(repo.FS, p)
		if err != nil {
			return err
		}
		return testsupport.Equal(string(raw), want)
	})
	sc.Step(`^acquired repository root matches "([^"]*)" below temporary directory$`, func(ctx context.Context, pattern string) error {
		relative, err := filepath.Rel(temp, repo.Root)
		if err != nil {
			return err
		}
		matched, err := filepath.Match(pattern, relative)
		testsupport.Log("repository root=%s relative=%s matched=%t", repo.Root, relative, matched)
		if err != nil {
			return err
		}
		return testsupport.Equal(matched, true)
	})
	sc.Step(`^archive calls equal "([^"]*)"$`, func(ctx context.Context, want string) error { return testsupport.Equal(strconv.Itoa(calls), want) })
	sc.Step(`^repository error contains "([^"]*)"$`, func(ctx context.Context, want string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), want) {
			return fmt.Errorf("error=%v want %s", failure, want)
		}
		return nil
	})
}
