package model

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// PathKind names the form a path inside a repository is written in.
type PathKind string

const (
	// OsAbsolute is a path from the OS filesystem root, e.g.
	// "/home/u/skills/a/b/c.md".
	OsAbsolute PathKind = "os-absolute"
	// RepoAbsolute is a path from the repository folder, written without a
	// "./" prefix, e.g. "a/b/c.md".
	RepoAbsolute PathKind = "repo-absolute"
	// FileRelative is a path from the folder of the file it is written in,
	// always starting with "./" or "../", e.g. "./c.md", "../b/c.md".
	FileRelative PathKind = "file-relative"
	// SkillRelative is a path from a skill's folder, e.g. "docs/c.md".
	SkillRelative PathKind = "skill-relative"
)

// DetectPathKind tells which form a path written in a file's text is in:
// "/..." (or a Windows drive "C:...") is OsAbsolute, "./..." or "../..." is
// FileRelative, anything else is RepoAbsolute. It never returns
// SkillRelative -- that form is never written in links.
func DetectPathKind(written string) PathKind {
	switch {
	case strings.HasPrefix(written, "/") || (len(written) > 1 && written[1] == ':'):
		return OsAbsolute
	case written == "." || written == ".." || strings.HasPrefix(written, "./") || strings.HasPrefix(written, "../"):
		return FileRelative
	default:
		return RepoAbsolute
	}
}

// PathInRepo is one path inside a repository, the single place that
// converts it between PathKind forms. It keeps the path RepoAbsolute and
// the repository folder's own OS path, which OsAbsolute needs. It never
// touches the filesystem.
type PathInRepo struct {
	repoDirOsPath string // "/home/u/skills" -- the repository folder in the OS
	pathInRepo    string // "a/b/c.md" -- clean, slash-separated, "." is the repository folder itself
}

// MakePathInRepo reads p written in form kind. baseDir is the RepoAbsolute
// folder a FileRelative p starts from (the folder of the file it is written
// in) or a SkillRelative p starts from (the skill's folder); other kinds
// ignore it. Fails with "path-escape" if p points outside the repository.
func MakePathInRepo(repoDirOsPath, p string, kind PathKind, baseDir string) (PathInRepo, error) {
	p = filepath.ToSlash(p)
	switch kind {
	case OsAbsolute:
		repoDir := strings.TrimSuffix(filepath.ToSlash(repoDirOsPath), "/")
		switch {
		case repoDir != "" && p == repoDir:
			p = "."
		case repoDir != "" && strings.HasPrefix(p, repoDir+"/"):
			p = strings.TrimPrefix(p, repoDir+"/")
		default:
			return PathInRepo{}, Problem("path-escape", p)
		}
	case RepoAbsolute:
	case FileRelative, SkillRelative:
		p = path.Join(baseDir, p)
	default:
		return PathInRepo{}, fmt.Errorf("unknown path kind %q", kind)
	}
	p = path.Clean(p)
	if !fs.ValidPath(p) {
		return PathInRepo{}, Problem("path-escape", p)
	}
	return PathInRepo{repoDirOsPath: repoDirOsPath, pathInRepo: p}, nil
}

// Path writes p in form kind. baseDir is the RepoAbsolute folder a
// FileRelative result starts from (the folder of the file it will be
// written in) or a SkillRelative result starts from (the skill's folder);
// other kinds ignore it. A FileRelative result always starts with "./" or
// "../", so it is never mistaken for RepoAbsolute.
func (p PathInRepo) Path(kind PathKind, baseDir string) (string, error) {
	switch kind {
	case OsAbsolute:
		return filepath.Join(p.repoDirOsPath, filepath.FromSlash(p.pathInRepo)), nil
	case RepoAbsolute:
		return p.pathInRepo, nil
	case FileRelative, SkillRelative:
		if baseDir == "" {
			baseDir = "."
		}
		rel, err := filepath.Rel(filepath.FromSlash(baseDir), filepath.FromSlash(p.pathInRepo))
		if err != nil {
			return "", err
		}
		rel = filepath.ToSlash(rel)
		if kind == FileRelative && rel != "." && rel != ".." && !strings.HasPrefix(rel, "../") {
			rel = "./" + strings.TrimPrefix(rel, "./")
		}
		return rel, nil
	default:
		return "", fmt.Errorf("unknown path kind %q", kind)
	}
}
