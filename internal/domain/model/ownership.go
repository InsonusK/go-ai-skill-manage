package model

import (
	"path"
	"strings"
)

// OwnsPath reports whether p belongs to a skill rooted at root (or the flat
// file main). A path inside a directory skill's root is owned even when no
// file has been read there yet.
func OwnsPath(main, root string, flat bool, p string) bool {
	if p == main || p == root {
		return true
	}
	return !flat && (root == "." || strings.HasPrefix(p, strings.TrimSuffix(root, "/")+"/"))
}

// RelativePath expresses p relative to a skill's own output directory:
// "SKILL.md" for the skill's root/main file, p unchanged for a "." root,
// otherwise p with the root prefix trimmed.
func RelativePath(main, root, p string) string {
	if p == main || p == root {
		return "SKILL.md"
	}
	if root == "." {
		return p
	}
	return strings.TrimPrefix(p, path.Clean(root)+"/")
}
