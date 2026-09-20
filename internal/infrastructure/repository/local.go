package repository

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Local struct{}

func (Local) Acquire(ctx context.Context, s model.SourceSpec) (*model.Repository, func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	absolute, err := filepath.Abs(s.Path)
	if err != nil {
		return nil, nil, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, nil, err
	}
	rootPath := absolute
	paths := s.Subpaths
	if !info.IsDir() {
		rootPath = filepath.Dir(absolute)
		paths = []string{filepath.Base(absolute)}
	} else if len(paths) == 0 {
		paths = []string{"."}
	}
	normalized := []string{}
	for _, p := range paths {
		if filepath.IsAbs(p) {
			p, err = filepath.Rel(rootPath, p)
			if err != nil {
				return nil, nil, err
			}
		}
		p = filepath.ToSlash(filepath.Clean(p))
		if !fs.ValidPath(p) || strings.Contains(p, "\\") {
			return nil, nil, fmt.Errorf("unsafe subpath %q", p)
		}
		normalized = append(normalized, p)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, nil, err
	}
	return &model.Repository{ID: "local:" + rootPath, Root: rootPath, FS: root.FS(), ScanPaths: normalized, Spec: s}, root.Close, nil
}
