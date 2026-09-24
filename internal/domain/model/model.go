// Package model defines transport-independent synchronization data,
// including the SkillCatalog registry and the OwnsPath/RelativePath
// ownership primitives it shares with it. Depends only on its own
// subpackage issues.
package model

import (
	"io/fs"
)

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

type OutputFile struct {
	Path string
	Data []byte
	Mode fs.FileMode
}
type Managed struct {
	Hash, Version            string
	Managed, Exists, HasMain bool
}
type Operation struct {
	Name, Action, Reason, Hash string
	Files                      []OutputFile
}
type TargetPlan struct {
	Target     Target
	Operations []Operation
	Shared     []OutputFile
}
type Result struct {
	Skills []string
	Plans  []TargetPlan
	DryRun bool
}
