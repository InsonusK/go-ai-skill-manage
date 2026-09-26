package filesystem

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Apply carries out plan in its target folder. It only executes what the
// plan decided (see planning.Plan), but guards the folder on its own:
//   - the plan is checked before anything is written: every skill folder
//     name is one plain path element, every file path stays inside its
//     folder, every written skill has the marker file -- without it the
//     next sync couldn't manage the folder;
//   - a folder is replaced or removed only if it has the marker, even when
//     the plan says so (the folder may have changed since it was planned);
//   - a skill folder is written into a temporary folder first and swapped
//     in by renames, so a failure leaves the old folder in place;
//   - a target folder reached through a symlink is refused.
//
// A skill's file contents and modes are read here, while writing.
func (Store) Apply(ctx context.Context, plan entity.TargetPlan) error {
	if err := validate(plan); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := safeTarget(plan.Target.Path); err != nil {
		return err
	}
	if err := os.MkdirAll(plan.Target.Path, 0o755); err != nil {
		return err
	}
	root, err := os.OpenRoot(plan.Target.Path)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, op := range plan.Operations {
		if err := ctx.Err(); err != nil {
			return err
		}
		if op.Action == entity.RemoveTarget {
			if err := requireManaged(root, op.Name); err != nil {
				return err
			}
			if err := root.RemoveAll(op.Name); err != nil {
				return err
			}
			continue
		}
		if err := replace(ctx, root, op); err != nil {
			return err
		}
	}
	return nil
}

func validate(plan entity.TargetPlan) error {
	names := map[string]bool{}
	for _, op := range plan.Operations {
		if !safeName(op.Name) || names[op.Name] {
			return fmt.Errorf("unsafe or duplicate skill folder name %q", op.Name)
		}
		names[op.Name] = true
		switch op.Action {
		case entity.RemoveTarget:
			continue
		case entity.CreateTarget, entity.UpdateTarget:
		default:
			return fmt.Errorf("unknown action %q for %s", op.Action, op.Name)
		}
		if op.Skill == nil {
			return fmt.Errorf("%s %s has no skill to write", op.Action, op.Name)
		}
		files, err := skillFiles(op.Skill)
		if err != nil {
			return err
		}
		marked := false
		for _, f := range files {
			if !safeFile(f.Path()) {
				return fmt.Errorf("unsafe output path %q in %s", f.Path(), op.Name)
			}
			marked = marked || f.Path() == model.Marker
		}
		if !marked {
			return fmt.Errorf("%s has no %s: a folder without it can't be managed later", op.Name, model.Marker)
		}
	}
	return nil
}

// skillFiles is the skill's marker file and all its other files.
func skillFiles(s *entity.TargetSkill) ([]*entity.TargetFile, error) {
	files, err := s.Files()
	if err != nil {
		return nil, err
	}
	return append([]*entity.TargetFile{s.MainFile()}, files...), nil
}

// safeName reports whether name is one plain path element.
func safeName(name string) bool { return safeFile(name) && !strings.Contains(name, "/") }

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
		return fmt.Errorf("unmanaged target %s: no %s", name, model.Marker)
	}
	return nil
}

// replace writes op's skill into a temporary folder and swaps it in for
// the skill's folder, keeping the old one until the swap succeeds.
func replace(ctx context.Context, root *os.Root, op entity.TargetOperation) (err error) {
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
	if err = root.Mkdir(stage, 0o755); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, root.RemoveAll(stage)) }()
	files, err := skillFiles(op.Skill)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err = ctx.Err(); err != nil {
			return err
		}
		content, err := f.Content()
		if err != nil {
			return err
		}
		mode, err := f.Mode()
		if err != nil {
			return err
		}
		if err = write(root, path.Join(stage, f.Path()), content, mode); err != nil {
			return err
		}
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

// write writes data at p with mode's permissions (0644 when it has none),
// through a temporary file renamed into place.
func write(root *os.Root, p string, data []byte, mode fs.FileMode) error {
	if err := root.MkdirAll(filepath.FromSlash(path.Dir(p)), 0o755); err != nil {
		return err
	}
	temp := filepath.FromSlash(path.Join(path.Dir(p), ".aism-write-"+rand.Text()))
	perm := mode.Perm()
	if perm == 0 {
		perm = entity.DefaultTargetFileMode
	}
	if err := root.WriteFile(temp, data, perm); err != nil {
		return err
	}
	// WriteFile's mode is cut by the umask; set the source's exactly.
	if err := root.Chmod(temp, perm); err != nil {
		_ = root.Remove(temp)
		return err
	}
	if err := root.Rename(temp, filepath.FromSlash(p)); err != nil {
		_ = root.Remove(temp)
		return err
	}
	return nil
}
