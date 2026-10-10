// Package feedback holds the rules of a feedback draft that need no I/O:
// what input is valid, how a draft is named and what issue it becomes.
package feedback

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
)

// labels maps a feedback kind to the issue label (GitHub's default ones).
var labels = map[model.FeedbackKind]string{
	model.FeedbackBug:         "bug",
	model.FeedbackImprovement: "enhancement",
}

// ValidKind reports whether kind is a known feedback kind.
func ValidKind(kind model.FeedbackKind) bool { _, ok := labels[kind]; return ok }

// CheckText rejects an empty title or body and a title of several lines.
func CheckText(title, body string) error {
	switch {
	case strings.TrimSpace(title) == "":
		return fmt.Errorf("feedback title is empty")
	case strings.ContainsAny(title, "\r\n"):
		return fmt.Errorf("feedback title must be one line")
	case strings.TrimSpace(body) == "":
		return fmt.Errorf("feedback body is empty")
	}
	return nil
}

// DraftID names a draft: date, skill and a slug of the title, so the file
// reads well in git log.
//
// Пример: (2026-09-27, "guide", "Broken anchor in SKILL.md") ->
// "2026-09-27-guide-broken-anchor-in-skill-md"; a title without latin
// letters or digits gives no slug: "2026-09-27-guide".
func DraftID(created time.Time, skill, title string) string {
	parts := []string{created.Format("2006-01-02")}
	for _, s := range []string{slug(skill), slug(title)} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "-")
}

const maxSlug = 48

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
			if b.Len() >= maxSlug {
				break
			}
			continue
		}
		dash = true
	}
	return b.String()
}

// Compose is the issue a draft becomes: its title, its body followed by
// where the skill comes from, and the label of its kind.
//
// Пример тела (скил guide из a/guide, коммит c0ffee, CLI 1.2.0):
//
//	<body>
//
//	---
//	Skill: `guide` (`a/guide` at commit `c0ffee`)
//	Source: github https://github.com/o/r (tree `main`)
//	Sent with ai-skill-manager 1.2.0
func Compose(draft model.FeedbackDraft, version string) model.NewIssue {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(draft.Body))
	b.WriteString("\n\n---\n")
	fmt.Fprintf(&b, "Skill: `%s` (`%s`", draft.Skill, draft.SkillPath)
	if draft.Commit != "" {
		fmt.Fprintf(&b, " at commit `%s`", draft.Commit)
	}
	b.WriteString(")\n")
	fmt.Fprintf(&b, "Source: %s %s", draft.Source.Type, draft.Source.Path)
	if draft.Source.Tree != "" {
		fmt.Fprintf(&b, " (tree `%s`)", draft.Source.Tree)
	}
	fmt.Fprintf(&b, "\nSent with ai-skill-manager %s\n", version)
	return model.NewIssue{Title: strings.TrimSpace(draft.Title), Body: b.String(), Labels: []string{labels[draft.Kind]}}
}

// Hash identifies exactly what would be sent where: the source and the
// composed issue. What the user confirmed is sent only if the hash still
// matches.
func Hash(source model.SourceKey, issue model.NewIssue) string {
	raw, err := json.Marshal(struct {
		Source model.SourceKey `json:"source"`
		Issue  model.NewIssue  `json:"issue"`
	}{source, issue})
	if err != nil {
		// Plain strings only: can't fail.
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
