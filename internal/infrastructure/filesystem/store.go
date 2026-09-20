package filesystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"os"
	"path/filepath"
)

type Store struct{}
type savedState struct {
	Hash    string `json:"hash"`
	Version string `json:"version"`
}

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
		if !entry.IsDir() {
			state[entry.Name()] = model.Managed{Exists: true}
			continue
		}
		name := entry.Name()
		item := model.Managed{Exists: true}
		raw, err := root.ReadFile(filepath.Join(name, model.Marker))
		if err == nil {
			item.Managed = true
			var saved savedState
			if json.Unmarshal(raw, &saved) == nil {
				item.Hash = saved.Hash
				item.Version = saved.Version
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		info, err := root.Stat(filepath.Join(name, "SKILL.md"))
		if err == nil {
			item.HasMain = info.Mode().IsRegular()
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		state[name] = item
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
