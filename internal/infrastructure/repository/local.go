package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

type Local struct{}

func (Local) Acquire(ctx context.Context, key model.SourceKey, _ model.AcquisitionOptions) (*entity.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	absoluteRootPath, err := filepath.Abs(key.Path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absoluteRootPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("source must be a directory")
	}

	root, err := os.OpenRoot(absoluteRootPath)
	if err != nil {
		return nil, err
	}
	repo := entity.MakeRepository(key, absoluteRootPath, root.FS(), root.Close)
	return repo, nil
}
