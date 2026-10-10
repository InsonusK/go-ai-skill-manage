package filesystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
)

var _ interfaces.MarkerReader = Store{}

// ReadMarker reads the marker (model.Marker) of skill's folder in target.
// No target folder, no skill folder or no regular marker file in it: an
// error wrapping interfaces.ErrSkillNotManaged, as is a skill entry that
// isn't a real folder (a symlink, a file). skill must be a single folder
// name.
func (Store) ReadMarker(ctx context.Context, target, skill string) (model.ManagedState, error) {
	var state model.ManagedState
	if !fs.ValidPath(skill) || skill == "." || strings.ContainsAny(skill, `/\`) {
		return state, fmt.Errorf("unsafe skill name %q", skill)
	}
	if err := safeTarget(target); err != nil {
		return state, err
	}
	notManaged := fmt.Errorf("%s in %s: %w", skill, target, interfaces.ErrSkillNotManaged)
	root, err := os.OpenRoot(target)
	if errors.Is(err, fs.ErrNotExist) {
		return state, notManaged
	}
	if err != nil {
		return state, err
	}
	defer root.Close()
	// As in Snapshot: only a real folder with a regular marker is managed.
	folder, err := root.Lstat(skill)
	if errors.Is(err, fs.ErrNotExist) {
		return state, notManaged
	}
	if err != nil {
		return state, err
	}
	if !folder.IsDir() {
		return state, notManaged
	}
	marker := filepath.Join(skill, model.Marker)
	info, err := root.Lstat(marker)
	if errors.Is(err, fs.ErrNotExist) {
		return state, notManaged
	}
	if err != nil {
		return state, err
	}
	if !info.Mode().IsRegular() {
		return state, notManaged
	}
	raw, err := root.ReadFile(marker)
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return state, fmt.Errorf("marker of %s in %s can't be read (written by an older version? run sync): %w", skill, target, err)
	}
	return state, nil
}
