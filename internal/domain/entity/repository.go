package entity

import (
	"errors"
	"io/fs"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

type Repository struct {
	Key      model.SourceKey
	RootPath string
	FS       fs.FS
	closers  []func() error
}

func MakeRepository(sourceKey model.SourceKey, rootPath string, fsys fs.FS, closeFunc func() error) *Repository {
	r := &Repository{Key: sourceKey, RootPath: rootPath, FS: fsys}
	if closeFunc != nil {
		r.AddCloser(closeFunc)
	}
	return r
}

func (r *Repository) IsExist(path string) bool {
	_, err := fs.Stat(r.FS, path)
	return !errors.Is(err, fs.ErrNotExist)
}

// Close runs every cleanup registered by the provider that built this
// Repository (temp directory removal, bounded root handles), in the order
// they were added.
func (r *Repository) Close() error {
	var err error
	for _, c := range r.closers {
		err = errors.Join(err, c())
	}
	return err
}

// AddCloser registers a cleanup function to run on Close, in call order.
func (r *Repository) AddCloser(c func() error) { r.closers = append(r.closers, c) }
