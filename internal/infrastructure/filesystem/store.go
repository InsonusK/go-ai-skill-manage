package filesystem

import (
	"context"
	"errors"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"os"
	"path/filepath"
)

// Store reads and writes target folders: Snapshot for planning, Apply to
// carry out a plan.
type Store struct{}

// Snapshot lists what the target folder holds, by entry name: every entry
// exists; a folder with a regular marker file (model.Marker) is managed.
// Files and symlinks are never managed. A missing target folder holds
// nothing.
func (Store) Snapshot(ctx context.Context, target string) (map[string]model.Managed, error) {
	state := map[string]model.Managed{}
	if err := safeTarget(target); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(target)
	if errors.Is(err, fs.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		item := model.Managed{Exists: true}
		if entry.IsDir() {
			marker, err := root.Lstat(filepath.Join(entry.Name(), model.Marker))
			switch {
			case err == nil:
				item.Managed = marker.Mode().IsRegular()
			case !errors.Is(err, fs.ErrNotExist):
				return nil, err
			}
		}
		state[entry.Name()] = item
	}
	return state, nil
}

// Reject symlinked target ancestors before opening the bounded os.Root.
func safeTarget(target string) error {
	absolute, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	for p := absolute; ; p = filepath.Dir(p) {
		info, err := os.Lstat(p)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsafe target symlink: %s", p)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}
