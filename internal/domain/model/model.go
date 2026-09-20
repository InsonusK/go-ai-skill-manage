// Package model defines transport-independent synchronization data.
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
type Repository struct {
	ID, Root  string
	FS        fs.FS
	ScanPaths []string
	Spec      SourceSpec
}
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
type Skill struct {
	Name, Main, Root string
	Flat             bool
	Repo             *Repository
	Document         Document
	Files            []File
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
