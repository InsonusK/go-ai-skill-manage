// Package model defines transport-independent synchronization data: the
// request (sources, targets, settings), source identity, paths inside a
// repository and parsed links. Depends only on its own subpackage issues.
package model

const Marker = ".ai-skills-managed"
const TransformVersion = "go-2"

// DefaultExcludeFromChecks is the global exclusion list used when the
// configuration doesn't set one: "examples" usually holds ready example
// code whose links often lead nowhere on purpose.
var DefaultExcludeFromChecks = []string{"examples"}

type SourceSpec struct {
	Type, Path, Tree, Name string
	Subpaths, Tags         []string
	// ExcludeFromChecks are top-level folders of this source's skills that
	// are loaded and copied with the skill but never checked (no
	// nested-skill, no link checks). Adds to Request.ExcludeFromChecks.
	ExcludeFromChecks []string
}

type AcquisitionOptions struct {
	TempDir string
}
type Target struct {
	Name, Path string
	Adapters   []string
}
type Request struct {
	Base                                       string
	TempDir                                    string
	Sources                                    []SourceSpec
	Targets                                    []Target
	DryRun, Force, RemoveOrphans, AddRelations bool
	Conflict                                   string
	// ExcludeFromChecks are top-level skill folders excluded from checks in
	// every source (see SourceSpec.ExcludeFromChecks).
	ExcludeFromChecks []string
}

// Managed is what a target folder holds under one name: whether there is
// an entry at all (a folder, a file, a symlink) and whether it is a folder
// this tool wrote (it has the Marker file).
type Managed struct {
	Exists, Managed bool
}
