package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"os"
	"path/filepath"
)

type Cloner interface {
	Clone(context.Context, string, string, string) error
}
type ArchiveFetcher interface {
	Fetch(context.Context, string, string, string) (string, error)
}
type Fetcher struct {
	Git     Cloner
	Archive ArchiveFetcher
}

func (f Fetcher) Acquire(ctx context.Context, s model.SourceSpec, options model.AcquisitionOptions) (*model.Repository, error) {
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
	tree := s.Tree
	if tree == "" {
		tree = "master"
	}
	cloneErr := f.Git.Clone(ctx, s.Path, tree, root)
	if cloneErr != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, _, err := GitHubURL(s.Path); err != nil {
			return nil, cloneErr
		}
		if err := os.RemoveAll(root); err != nil {
			return nil, err
		}
		root, err = f.Archive.Fetch(ctx, s.Path, tree, filepath.Join(temp, "archive"))
		if err != nil {
			return nil, fmt.Errorf("source acquisition: %w", errors.Join(cloneErr, err))
		}
	}
	local := s
	local.Path = root
	repo, err := (Local{}).Acquire(ctx, local, options)
	if err != nil {
		return nil, err
	}
	repo.ID = s.Path + "@" + tree
	repo.AddCloser(func() error { return os.RemoveAll(temp) })
	failed = false
	return repo, nil
}
