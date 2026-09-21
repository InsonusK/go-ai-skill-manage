// Package model defines transport-independent synchronization data,
// including the Catalog and SkillMap/SkillEntry registries and the
// OwnsPath/RelativePath ownership primitives they share. Zero outward
// dependencies.
package model

import (
	"errors"
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

// SourceKey identifies what to fetch (type/path/tree), excluding selection
// fields (Subpaths/Tags/Name) that vary per SourceSpec even when they point at
// the same underlying source. Used by SourceManager to dedup acquisitions.
type SourceKey struct{ Type, Path, Tree string }

func (s SourceSpec) Key() SourceKey { return SourceKey{Type: s.Type, Path: s.Path, Tree: s.Tree} }

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
type Repository struct {
	ID, Root    string
	FS          fs.FS
	SingleFile  string   // relative path, set when the source itself is one flat skill file
	SkipFolders []string // this source's skip_folder, baked in once at first acquisition
	closers     []func() error
}

// Close runs every cleanup registered by the provider that built this
// Repository (temp directory removal, bounded root handles), in the order
// they were added.
func (r *Repository) Close() error {
	var err error
	for _, c := range r.closers {
		err = errors.Join(err, c())
	}
	return err
}

// AddCloser registers a cleanup function to run on Close, in call order.
func (r *Repository) AddCloser(c func() error) { r.closers = append(r.closers, c) }

type Document struct {
	Properties     map[string]any
	Metadata       map[string]any
	Body           string
	HasFrontmatter bool
}
type File struct {
	Path  string
	Data  []byte
	Mode  fs.FileMode
	Links []Link
}

// SkillFormat names which of the three on-disk skill layouts a Skill was
// found in.
type SkillFormat string

const (
	// FlatSkill is a single "{name}.skill.md" file with no directory.
	FlatSkill SkillFormat = "flat"
	// HumanDirSkill is a "{name}.skill/" directory whose marker file is
	// "{name}.skill.md".
	HumanDirSkill SkillFormat = "human-dir"
	// AgentDirSkill is a "{name}.skill/" directory whose marker file is
	// "SKILL.md".
	AgentDirSkill SkillFormat = "agent-dir"
)

type Skill struct {
	Name, Main, Root string      // Main/Root are repo-relative; Root is "" for FlatSkill (N/A)
	Format           SkillFormat
	Repo             *Repository
	Document         Document
	MainFile         File   // the skill's own file; Path is always "SKILL.md", Data always populated
	Files            []File // nested files only (never the main file); Path is skill-relative
}

func (s *Skill) Key() string { return s.Repo.ID + "\x00" + s.Main }

type Link struct {
	Start, End                        int
	Raw, Text, Path, Fragment, Format string
	Image                             bool
	Target                            string
}
type Issue struct{ Code, Skill, File, Link, Message string }

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
