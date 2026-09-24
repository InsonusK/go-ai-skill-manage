package validators

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model/issues"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/validator"
)

// SubpathValidator checks that every subpath stays inside its source,
// without looking at the filesystem: whether it exists is only known once
// the source is acquired.
type SubpathValidator struct{}

var _ ConfigValidator = SubpathValidator{}

func (SubpathValidator) Name() string { return "subpath-validator" }

func (SubpathValidator) DependsOn() []validator.Dependency { return nil }

// Validate reports "unsafe-subpath" for a subpath leading out of its
// source.
//
// Примеры (local-источник "/project/skills"):
//   - "a/b", "./a", "a/../b", "/project/skills/a" -> ok
//   - "../x", "a/../../x" -> выходит вверх
//   - `a\b` -> обратный слэш
//   - "/project/other" -> абсолютный путь вне источника
//   - "/x" в github-источнике -> абсолютный путь в github-источнике
func (SubpathValidator) Validate(ctx context.Context, req model.Request) []issues.ConfigIssue {
	var problems []issues.ConfigIssue
	for i, spec := range req.Sources {
		for j, p := range spec.Subpaths {
			if reason := unsafeSubpath(spec, p); reason != "" {
				problems = append(problems, issues.ConfigIssue{Code: "unsafe-subpath", Source: spec.Key().String(), Setting: fmt.Sprintf("sources[%d].subpath[%d]", i, j), Message: fmt.Sprintf("subpath %q %s", p, reason)})
			}
		}
	}
	return problems
}

// unsafeSubpath returns why p leads out of spec's source, or "" if it
// doesn't.
func unsafeSubpath(spec model.SourceSpec, p string) string {
	if strings.Contains(p, "\\") {
		return "contains a backslash"
	}
	if filepath.IsAbs(p) {
		if spec.Type != "local" {
			return "is absolute in a " + spec.Type + " source"
		}
		if !within(filepath.Clean(p), filepath.Clean(spec.Path)) {
			return "is outside the source " + spec.Path
		}
		return ""
	}
	if !fs.ValidPath(path.Clean(p)) {
		return "leads out of the source"
	}
	return ""
}

// within reports whether p is dir or lies below it.
func within(p, dir string) bool {
	return p == dir || strings.HasPrefix(p, strings.TrimRight(dir, string(filepath.Separator))+string(filepath.Separator))
}
