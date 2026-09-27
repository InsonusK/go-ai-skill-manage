package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Cloner clones (url, tree) into dest and returns the cloned commit.
type Cloner interface {
	Clone(ctx context.Context, url, tree, dest string) (string, error)
}

// ArchiveFetcher extracts (url, tree) below dest and returns the tree's
// root folder and its commit ("" when the archive doesn't name it).
type ArchiveFetcher interface {
	Fetch(ctx context.Context, url, tree, dest string) (string, string, error)
}
type Fetcher struct {
	Git     Cloner
	Archive ArchiveFetcher
}

func (f Fetcher) Acquire(ctx context.Context, key model.SourceKey, options model.AcquisitionOptions) (*entity.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	temp, err := os.MkdirTemp(options.TempDir, "aism-source-")
	if err != nil {
		return nil, err
	}
	failed := true
	defer func() {
		if failed {
			_ = os.RemoveAll(temp)
		}
	}()
	root := filepath.Join(temp, "repo")
	tree := key.Tree
	if tree == "" {
		tree = "master"
	}
	commit, cloneErr := f.Git.Clone(ctx, key.Path, tree, root)
	if cloneErr != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, _, err := GitHubURL(key.Path); err != nil {
			return nil, cloneErr
		}
		if err := os.RemoveAll(root); err != nil {
			return nil, err
		}
		root, commit, err = f.Archive.Fetch(ctx, key.Path, tree, filepath.Join(temp, "archive"))
		if err != nil {
			return nil, fmt.Errorf("source acquisition: %w", errors.Join(cloneErr, err))
		}
	}
	local := key
	local.Path = root
	repo, err := (Local{}).Acquire(ctx, local, options)
	if err != nil {
		return nil, err
	}
	repo.Key = key
	repo.Commit = commit
	repo.AddCloser(func() error { return os.RemoveAll(temp) })
	failed = false
	return repo, nil
}
