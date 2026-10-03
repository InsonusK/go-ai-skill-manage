package filesystem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/interfaces"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"go.yaml.in/yaml/v3"
)

// FeedbackDrafts keeps feedback drafts as markdown files "<id>.md" in Dir:
// YAML frontmatter with everything but the text, then "# <title>" and the
// body. The file is meant to be read and edited by the user and
// committed.
//
// Пример:
//
//	---
//	status: draft
//	kind: bug
//	skill: guide
//	source:
//	  type: github
//	  path: https://github.com/o/r
//	  tree: main
//	commit: c0ffee
//	skill_path: a/guide
//	created_at: 2026-09-27T10:00:00Z
//	---
//
//	# Broken anchor
//
//	The anchor #x is missing.
type FeedbackDrafts struct{ Dir string }

var _ interfaces.FeedbackDrafts = FeedbackDrafts{}

// draftID: what DraftID produces -- and nothing that could leave Dir.
var draftID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type draftFront struct {
	Status    model.FeedbackStatus `yaml:"status"`
	Kind      model.FeedbackKind   `yaml:"kind"`
	Skill     string               `yaml:"skill"`
	Source    draftSource          `yaml:"source"`
	Commit    string               `yaml:"commit,omitempty"`
	SkillPath string               `yaml:"skill_path"`
	CreatedAt time.Time            `yaml:"created_at"`
	IssueURL  string               `yaml:"issue_url,omitempty"`
	SentAt    *time.Time           `yaml:"sent_at,omitempty"`
}

type draftSource struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	Tree string `yaml:"tree,omitempty"`
}

// Create writes draft as "<id>.md" or, when taken, "<id>-2.md", "<id>-3.md"...
func (d FeedbackDrafts) Create(ctx context.Context, id string, draft model.FeedbackDraft) (string, error) {
	if !draftID.MatchString(id) {
		return "", fmt.Errorf("invalid feedback id %q", id)
	}
	if err := os.MkdirAll(d.Dir, 0o755); err != nil {
		return "", err
	}
	for n := 1; ; n++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		free := id
		if n > 1 {
			free = id + "-" + strconv.Itoa(n)
		}
		draft.ID = free
		content, err := encodeDraft(draft)
		if err != nil {
			return "", err
		}
		file, err := os.OpenFile(d.path(free), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, writeErr := file.Write(content)
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return "", err
		}
		return free, nil
	}
}

func (d FeedbackDrafts) Load(ctx context.Context, id string) (model.FeedbackDraft, error) {
	if !draftID.MatchString(id) {
		return model.FeedbackDraft{}, fmt.Errorf("invalid feedback id %q", id)
	}
	raw, err := os.ReadFile(d.path(id))
	if errors.Is(err, fs.ErrNotExist) {
		return model.FeedbackDraft{}, fmt.Errorf("no feedback %s in %s", id, d.Dir)
	}
	if err != nil {
		return model.FeedbackDraft{}, err
	}
	draft, err := decodeDraft(raw)
	if err != nil {
		return draft, fmt.Errorf("feedback %s: %w", id, err)
	}
	draft.ID = id
	return draft, nil
}

// List returns the ids of the "<id>.md" files in Dir, sorted; other files
// and a missing Dir give none.
func (d FeedbackDrafts) List(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(d.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		id, ok := strings.CutSuffix(entry.Name(), ".md")
		if ok && entry.Type().IsRegular() && draftID.MatchString(id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Save overwrites the existing file of draft.
func (d FeedbackDrafts) Save(ctx context.Context, draft model.FeedbackDraft) error {
	if !draftID.MatchString(draft.ID) {
		return fmt.Errorf("invalid feedback id %q", draft.ID)
	}
	content, err := encodeDraft(draft)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(d.path(draft.ID), os.O_WRONLY|os.O_TRUNC, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no feedback %s in %s", draft.ID, d.Dir)
	}
	if err != nil {
		return err
	}
	_, writeErr := file.Write(content)
	return errors.Join(writeErr, file.Close())
}

func (d FeedbackDrafts) path(id string) string { return filepath.Join(d.Dir, id+".md") }

func encodeDraft(draft model.FeedbackDraft) ([]byte, error) {
	front := draftFront{
		Status: draft.Status, Kind: draft.Kind, Skill: draft.Skill,
		Source: draftSource{Type: draft.Source.Type, Path: draft.Source.Path, Tree: draft.Source.Tree},
		Commit: draft.Commit, SkillPath: draft.SkillPath, CreatedAt: draft.CreatedAt.UTC().Truncate(time.Second), IssueURL: draft.IssueURL,
	}
	if !draft.SentAt.IsZero() {
		at := draft.SentAt.UTC().Truncate(time.Second)
		front.SentAt = &at
	}
	var b bytes.Buffer
	b.WriteString("---\n")
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(front); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	b.WriteString("---\n\n# ")
	b.WriteString(strings.TrimSpace(draft.Title))
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(draft.Body))
	b.WriteString("\n")
	return b.Bytes(), nil
}

func decodeDraft(raw []byte) (model.FeedbackDraft, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return model.FeedbackDraft{}, fmt.Errorf("no frontmatter: the file must start with ---")
	}
	head, text, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return model.FeedbackDraft{}, fmt.Errorf("frontmatter isn't closed with ---")
	}
	var front draftFront
	if err := yaml.Unmarshal([]byte(head), &front); err != nil {
		return model.FeedbackDraft{}, fmt.Errorf("frontmatter: %w", err)
	}
	text = strings.TrimLeft(text, "\n")
	heading, body, _ := strings.Cut(text, "\n")
	title, ok := strings.CutPrefix(heading, "# ")
	if !ok {
		return model.FeedbackDraft{}, fmt.Errorf("the text must start with the title as \"# <title>\"")
	}
	draft := model.FeedbackDraft{
		Status: front.Status, Kind: front.Kind, Skill: front.Skill,
		Source: model.SourceKey{Type: front.Source.Type, Path: front.Source.Path, Tree: front.Source.Tree},
		Commit: front.Commit, SkillPath: front.SkillPath, CreatedAt: front.CreatedAt, IssueURL: front.IssueURL,
		Title: strings.TrimSpace(title), Body: strings.TrimSpace(body),
	}
	if front.SentAt != nil {
		draft.SentAt = *front.SentAt
	}
	return draft, nil
}
