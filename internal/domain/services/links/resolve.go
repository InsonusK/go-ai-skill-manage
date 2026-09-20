package links

import (
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

func Resolve(repo *model.Repository, from, raw string, knownOwner func(string) bool) (string, error) {
	p := strings.ReplaceAll(raw, "\\", "/")
	switch {
	case strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../"):
		p = path.Join(path.Dir(from), p)
	case strings.HasPrefix(p, "/") || (len(p) > 1 && p[1] == ':'):
		root := strings.TrimSuffix(filepath.ToSlash(repo.Root), "/")
		if p == root {
			p = "."
		} else if strings.HasPrefix(p, root+"/") {
			p = strings.TrimPrefix(p, root+"/")
		} else {
			return "", model.Problem("path-escape", raw)
		}
	default:
		p = path.Clean(p)
	}
	if !fs.ValidPath(p) {
		return "", model.Problem("path-escape", raw)
	}
	if _, err := fs.Stat(repo.FS, p); err == nil {
		return p, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", model.Problem("source-read", err.Error())
	}
	if _, err := fs.Stat(repo.FS, p+".md"); err == nil {
		return p + ".md", nil
	}
	// Python resolves selected skill ownership before testing external-file existence.
	if knownOwner != nil && knownOwner(p) {
		return p, nil
	}
	return "", model.Problem("missing-link", raw)
}
