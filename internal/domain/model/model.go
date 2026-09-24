// Package model defines transport-independent synchronization data,
// including the SkillCatalog registry and the OwnsPath/RelativePath
// ownership primitives it shares with it. Zero outward dependencies.
package model

import (
	"fmt"
	"io/fs"
	"strings"
)

const Marker = ".ai-skills-managed"
const TransformVersion = "go-2"

type SourceSpec struct {
	Type, Path, Tree, Name      string
	Subpaths, Tags, SkipFolders []string
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
	LinkSkipFolders                            []string
}

// Issue is one problem found while loading or validating skills, with
// enough context to report it grouped by skill and file.
type Issue struct {
	// Code is the kind of problem, e.g. "nested-skill", "missing-link-target".
	Code string
	// Source is the source the skill comes from (model.SourceKey.String()).
	Source string
	// Skill is the skill's name; SkillPath is its folder (or, for a flat
	// skill, its marker file) from the repository folder.
	Skill, SkillPath string
	// File is the file the problem is in; Link is the link's text as
	// written, when the problem is a link.
	File, Link string
	Message    string
}

func (e Issue) Error() string {
	context := strings.Trim(strings.Join([]string{e.Skill, e.File, e.Link}, " "), " ")
	return fmt.Sprintf("%s: %s: %s", e.Code, context, e.Message)
}

type Issues []Issue

func (e Issues) Error() string {
	var out []string
	for _, i := range e {
		out = append(out, i.Error())
	}
	return strings.Join(out, "\n")
}
func Problem(code, message string) error { return Issue{Code: code, Message: message} }

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
