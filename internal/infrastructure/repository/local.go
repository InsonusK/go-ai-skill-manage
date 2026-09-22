package repository

import (
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"os"
	"path/filepath"
)

type Local struct{}

func (Local) Acquire(ctx context.Context, key model.SourceKey, _ model.AcquisitionOptions) (*model.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(key.Path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, err
	}
	rootPath := absolute
	singleFile := ""
	if !info.IsDir() {
		rootPath = filepath.Dir(absolute)
		singleFile = filepath.Base(absolute)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	// TODO(discovery step 3): SkipFolders used to come from SourceSpec here
	// (repo.SkipFolders = s.SkipFolders); SourceProvider now only gets a
	// SourceKey, so it's never set. See interfaces.SourceProvider's doc
	// comment for the full note.
	repo := &model.Repository{ID: "local:" + rootPath, Root: rootPath, FS: root.FS(), SingleFile: singleFile}
	repo.AddCloser(root.Close)
	return repo, nil
}
