package entity

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// ErrSkillNotCached reports that a cache-only SkillResolver lookup found no
// skill for the requested path among the skills already loaded.
var ErrSkillNotCached = errors.New("skill is not in the catalog cache")

// SkillResolver finds skills by a path inside a source's repository.
// sourcing.SkillCatalog implements it; entity declares it because sourcing
// imports entity. "ByPath" finds skills at or below the path, "ByPathUp"
// finds the one skill whose folder holds the path. "Get" only looks at
// skills already loaded and fails with ErrSkillNotCached otherwise;
// "TryGetOrFetch" also loads a missing skill when the resolver is set to
// add relations, and behaves like "Get" when it is not.
type SkillResolver interface {
	GetByPath(ctx context.Context, key model.SourceKey, p string) ([]*Skill, error)
	GetByPathUp(ctx context.Context, key model.SourceKey, p string) (*Skill, error)
	TryGetOrFetchByPath(ctx context.Context, key model.SourceKey, p string) ([]*Skill, error)
	TryGetOrFetchByPathUp(ctx context.Context, key model.SourceKey, p string) (*Skill, error)
}

// webPrefixes mark a written link path as a web resource rather than a path
// inside the repository.
var webPrefixes = []string{"http://", "https://", "mailto:", "ftp://", "file://"}

// Link is one link found in a File's text. Only links inside the file's
// own repository and web links are supported.
type Link struct {
	// Start and End are the [Start, End) byte range of the link in the
	// file content; Raw is that range's text.
	Start, End int
	Raw        string
	// Text, WrittenPath, Fragment -- the link's parts:
	//   - markdown [text](path#fragment)
	//   - wikilink [[path#fragment|text]]
	// WrittenPath is exactly as written, e.g. "./guide", "a/b.md",
	// "https://x"; Fragment keeps its leading "#".
	Text, WrittenPath, Fragment string
	// Format is the link syntax, "markdown" or "wikilink".
	Format string
	// Image is true for the "!" image form of the link.
	Image bool
	// External is true when WrittenPath is a web resource (http://,
	// https://, mailto:, ftp://, file://), not a path in the repository.
	External bool

	file   *File
	target *model.PathInRepo // cached by resolveTarget
}

// MakeLink builds the Link parsed from file's content: Raw is read from
// that content, External from the written path.
func MakeLink(file *File, parsed model.ParsedLink) (*Link, error) {
	content, err := file.Content()
	if err != nil {
		return nil, err
	}
	if parsed.Start < 0 || parsed.End > len(content) || parsed.Start >= parsed.End {
		return nil, model.Issue{Code: "invalid-link-span", Message: fmt.Sprintf("span [%d,%d) outside content of length %d", parsed.Start, parsed.End, len(content))}
	}
	external := false
	lower := strings.ToLower(parsed.Path)
	for _, prefix := range webPrefixes {
		if strings.HasPrefix(lower, prefix) {
			external = true
			break
		}
	}
	return &Link{
		Start:       parsed.Start,
		End:         parsed.End,
		Raw:         string(content[parsed.Start:parsed.End]),
		Text:        parsed.Text,
		WrittenPath: parsed.Path,
		Fragment:    parsed.Fragment,
		Format:      parsed.Format,
		Image:       parsed.Image,
		External:    external,
		file:        file,
	}, nil
}

// File returns the file the link is written in.
func (l *Link) File() *File { return l.file }

// Path returns the path of the file the link points to, in form kind.
// FileRelative starts from the folder of the file the link is written in.
// SkillRelative starts from the folder of the skill holding the target,
// found through resolver (see Skill); other kinds don't use resolver. Fails
// for a web link.
func (l *Link) Path(ctx context.Context, kind model.PathKind, resolver SkillResolver) (string, error) {
	target, err := l.resolveTarget()
	if err != nil {
		return "", err
	}
	switch kind {
	case model.FileRelative:
		from, err := l.file.Path(model.RepoAbsolute)
		if err != nil {
			return "", err
		}
		return target.Path(kind, path.Dir(from))
	case model.SkillRelative:
		skill, err := l.Skill(ctx, resolver)
		if err != nil {
			return "", err
		}
		return target.Path(kind, skill.SkillDirPath)
	default:
		return target.Path(kind, "")
	}
}

// Skill returns the skill whose folder holds the file the link points to,
// via resolver.TryGetOrFetchByPathUp -- so it is loaded only when resolver
// adds relations, otherwise it must already be loaded (ErrSkillNotCached).
// Fails for a web link.
func (l *Link) Skill(ctx context.Context, resolver SkillResolver) (*Skill, error) {
	target, err := l.resolveTarget()
	if err != nil {
		return nil, err
	}
	p, err := target.Path(model.RepoAbsolute, "")
	if err != nil {
		return nil, err
	}
	return resolver.TryGetOrFetchByPathUp(ctx, l.file.skill.Repo.Key, p)
}

// resolveTarget turns WrittenPath into the repository path it points to,
// once: a path is read in the form DetectPathKind finds ("./x" from the
// file's folder, "a/x" from the repository folder, "/x" from the OS root),
// an empty one (a "#fragment"-only link) points to the file itself, and a
// path with the ".md" left out (a/b/c meaning a/b/c.md) gets it written
// explicitly. Fails for a web link, a path leaving the repository
// ("path-escape"), or a target that doesn't exist ("missing-link-target").
func (l *Link) resolveTarget() (*model.PathInRepo, error) {
	if l.target != nil {
		return l.target, nil
	}
	if l.External {
		return nil, model.Issue{Code: "web-link", Link: l.Raw, Message: "a web link has no path in the repository"}
	}
	repo := l.file.skill.Repo
	from, err := l.file.Path(model.RepoAbsolute)
	if err != nil {
		return nil, err
	}
	written := strings.ReplaceAll(l.WrittenPath, "\\", "/")
	if written == "" {
		written = from
	}
	target, err := model.MakePathInRepo(repo.RootPath, written, model.DetectPathKind(written), path.Dir(from))
	if err != nil {
		return nil, model.Issue{Code: "path-escape", Link: l.Raw, Message: err.Error()}
	}
	p, err := target.Path(model.RepoAbsolute, "")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(repo.FS, p); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if _, err := fs.Stat(repo.FS, p+".md"); err != nil {
			return nil, model.Issue{Code: "missing-link-target", Link: l.Raw, File: p, Message: "link target does not exist"}
		}
		if target, err = model.MakePathInRepo(repo.RootPath, p+".md", model.RepoAbsolute, ""); err != nil {
			return nil, err
		}
	}
	l.target = &target
	return l.target, nil
}
