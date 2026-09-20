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

func (f Fetcher) Acquire(ctx context.Context, s model.SourceSpec, options model.AcquisitionOptions) (*model.Repository, func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	temp, err := os.MkdirTemp(options.TempDir, "aism-source-")
	if err != nil {
		return nil, nil, err
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
			return nil, nil, err
		}
		if _, _, err := GitHubURL(s.Path); err != nil {
			return nil, nil, cloneErr
		}
		if err := os.RemoveAll(root); err != nil {
			return nil, nil, err
		}
		root, err = f.Archive.Fetch(ctx, s.Path, tree, filepath.Join(temp, "archive"))
		if err != nil {
			return nil, nil, fmt.Errorf("source acquisition: %w", errors.Join(cloneErr, err))
		}
	}
	local := s
	local.Path = root
	repo, close, err := (Local{}).Acquire(ctx, local)
	if err != nil {
		return nil, nil, err
	}
	repo.ID = s.Path + "@" + tree
	repo.Spec = s
	failed = false
	return repo, func() error { return errors.Join(close(), os.RemoveAll(temp)) }, nil
}

type Provider struct {
	Local  Local
	Remote Fetcher
}

func (p Provider) Acquire(ctx context.Context, s model.SourceSpec, options model.AcquisitionOptions) (*model.Repository, func() error, error) {
	switch s.Type {
	case "local":
		return p.Local.Acquire(ctx, s)
	case "github":
		return p.Remote.Acquire(ctx, s, options)
	default:
		return nil, nil, fmt.Errorf("unknown source type %q", s.Type)
	}
}
