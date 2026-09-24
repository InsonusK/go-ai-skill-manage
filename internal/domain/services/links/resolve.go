package links

import (
	"errors"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// Deferred: everything here decides where a found link leads inside the
// repository and whether that target belongs to the current catalog -- not
// what the link is (that is link_parser's job) nor where it is in the text
// (LinkFactory's). Only relations.Expander uses it, and relations is not
// wired to the new SkillCatalog yet (see AGENTS.md); revisit together.

// InSkippedFolder reports whether file (the file a link is written in) lies
// under one of the skip folders (e.g. "examples"), whose links are not
// followed.
func InSkippedFolder(file string, skip []string) bool {
	for _, segment := range strings.Split(strings.ReplaceAll(file, "\\", "/"), "/") {
		for _, folder := range skip {
			if segment == folder {
				return true
			}
		}
	}
	return false
}

// Resolve turns a link path written in from into a repository-relative
// path: "./" and "../" are relative to from's folder, an absolute path must
// lie under repo.RootPath, and a bare path is relative to the repository
// root. A missing target is retried with ".md" appended, then accepted if
// knownOwner says it lies inside an already selected skill.
func Resolve(repo *entity.Repository, from, raw string, knownOwner func(string) bool) (string, error) {
	p := strings.ReplaceAll(raw, "\\", "/")
	switch {
	case strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../"):
		p = path.Join(path.Dir(from), p)
	case strings.HasPrefix(p, "/") || (len(p) > 1 && p[1] == ':'):
		root := strings.TrimSuffix(filepath.ToSlash(repo.RootPath), "/")
		if p == root {
			p = "."
		} else if strings.HasPrefix(p, root+"/") {
			p = strings.TrimPrefix(p, root+"/")
		} else {
			return "", issues.Problem("path-escape", raw)
		}
	default:
		p = path.Clean(p)
	}
	if !fs.ValidPath(p) {
		return "", issues.Problem("path-escape", raw)
	}
	if _, err := fs.Stat(repo.FS, p); err == nil {
		return p, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", issues.Problem("source-read", err.Error())
	}
	if _, err := fs.Stat(repo.FS, p+".md"); err == nil {
		return p + ".md", nil
	}
	// Python resolves selected skill ownership before testing external-file existence.
	if knownOwner != nil && knownOwner(p) {
		return p, nil
	}
	return "", issues.Problem("missing-link", raw)
}
