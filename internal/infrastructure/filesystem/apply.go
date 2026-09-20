package filesystem

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/discovery"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (Store) Apply(ctx context.Context, plan model.TargetPlan) error {
	if err := validate(plan); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeTarget(plan.Target.Path); err != nil {
		return err
	}
	if err := os.MkdirAll(plan.Target.Path, 0755); err != nil {
		return err
	}
	root, err := os.OpenRoot(plan.Target.Path)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, file := range plan.Shared {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := write(root, file); err != nil {
			return err
		}
	}
	for _, op := range plan.Operations {
		if err := ctx.Err(); err != nil {
			return err
		}
		switch op.Action {
		case "skip":
			continue
		case "remove":
			if err := requireManaged(root, op.Name); err != nil {
				return err
			}
			if err := root.RemoveAll(op.Name); err != nil {
				return err
			}
		default:
			if err := replace(root, op); err != nil {
				return err
			}
		}
	}
	return nil
}
func validate(plan model.TargetPlan) error {
	names := map[string]bool{}
	for _, op := range plan.Operations {
		if !discovery.ValidName(op.Name) || names[op.Name] {
			return fmt.Errorf("unsafe or duplicate skill name %q", op.Name)
		}
		names[op.Name] = true
		if op.Action != "create" && op.Action != "update" && op.Action != "skip" && op.Action != "remove" {
			return fmt.Errorf("unknown action %q", op.Action)
		}
		for _, file := range op.Files {
			if !safeFile(file.Path) || file.Path == model.Marker {
				return fmt.Errorf("unsafe output path %q", file.Path)
			}
		}
	}
	for _, file := range plan.Shared {
		if !safeFile(file.Path) || !strings.HasPrefix(file.Path, "files/") {
			return fmt.Errorf("unsafe shared path %q", file.Path)
		}
	}
	return nil
}
func safeFile(p string) bool { return p != "." && fs.ValidPath(p) && !strings.ContainsAny(p, "\\:") }
func requireManaged(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("unsafe target entry %s", name)
	}
	marker, err := root.Lstat(filepath.Join(name, model.Marker))
	if err != nil || !marker.Mode().IsRegular() {
		return fmt.Errorf("unmanaged target %s", name)
	}
	return nil
}
func replace(root *os.Root, op model.Operation) (err error) {
	exists := false
	if _, e := root.Lstat(op.Name); e == nil {
		exists = true
		if e = requireManaged(root, op.Name); e != nil {
			return e
		}
	} else if !errors.Is(e, fs.ErrNotExist) {
		return e
	}
	stage := ".aism-stage-" + rand.Text()
	if err = root.Mkdir(stage, 0755); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.RemoveAll(stage)) }()
	for _, f := range op.Files {
		f.Path = path.Join(stage, f.Path)
		if err = write(root, f); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(savedState{Hash: op.Hash, Version: model.TransformVersion})
	if err != nil {
		return err
	}
	if err = root.WriteFile(filepath.Join(stage, model.Marker), raw, 0644); err != nil {
		return err
	}
	backup := ".aism-backup-" + rand.Text()
	if exists {
		if err = root.Rename(op.Name, backup); err != nil {
			return err
		}
	}
	if err = root.Rename(stage, op.Name); err != nil {
		if exists {
			err = errors.Join(err, root.Rename(backup, op.Name))
		}
		return err
	}
	if exists {
		return root.RemoveAll(backup)
	}
	return nil
}
func write(root *os.Root, f model.OutputFile) error {
	if f.Mode.IsDir() {
		return root.MkdirAll(filepath.FromSlash(f.Path), 0755)
	}
	if err := root.MkdirAll(filepath.FromSlash(path.Dir(f.Path)), 0755); err != nil {
		return err
	}
	temp := filepath.FromSlash(path.Join(path.Dir(f.Path), ".aism-write-"+rand.Text()))
	mode := f.Mode.Perm()
	if mode == 0 {
		mode = 0644
	}
	if err := root.WriteFile(temp, f.Data, mode); err != nil {
		return err
	}
	if err := root.Rename(temp, filepath.FromSlash(f.Path)); err != nil {
		_ = root.Remove(temp)
		return err
	}
	return nil
}
