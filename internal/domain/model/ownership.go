package model

import (
	"path"
)

// NestedRepoPath expresses a nested (non-main) file's skill-relative Path
// back in repo-relative terms -- for link resolution's "from" parameter, or
// an OriginalKey. Only valid for HumanDirSkill/AgentDirSkill; a FlatSkill
// never has nested Files.
func NestedRepoPath(root, relative string) string { return path.Join(root, relative) }
