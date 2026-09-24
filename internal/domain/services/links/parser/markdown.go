package link_parser

import (
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// MarkdownParser handles "[label](path#fragment)" links and their
// "![alt](path)" image form. The label may hold balanced brackets, so a
// linked image "[![alt](img)](path)" is found as two links -- the outer one
// and the image inside its label. Titles ("[a](b "title")") and
// angle-bracket targets are not recognized.
type MarkdownParser struct{}

var _ LinkParser = MarkdownParser{}

func (MarkdownParser) Find(content string) []Span {
	out := []Span{}
	for i := 0; i < len(content); i++ {
		if _, _, end, ok := markdownAt(content, i); ok {
			start := i
			if i > 0 && content[i-1] == '!' {
				start--
			}
			out = append(out, Span{Start: start, End: end})
		}
	}
	return out
}

func (MarkdownParser) Parse(raw string) (model.ParsedLink, error) {
	open := 0
	if strings.HasPrefix(raw, "!") {
		open = 1
	}
	label, target, end, ok := markdownAt(raw, open)
	if !ok || end != len(raw) {
		return model.ParsedLink{}, invalidLink(raw, "markdown")
	}
	return newLink(raw, label, target, "markdown"), nil
}

// markdownAt matches a markdown link whose "[" is at s[open]: a label with
// balanced brackets, then "(target)" where target has no whitespace, ")" or
// '"'. It returns the label, the target and the end of the link.
func markdownAt(s string, open int) (label, target string, end int, ok bool) {
	if open >= len(s) || s[open] != '[' {
		return "", "", 0, false
	}
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}
			rest := s[i+1:]
			if !strings.HasPrefix(rest, "(") {
				return "", "", 0, false
			}
			j := strings.IndexAny(rest[1:], " \t\n\f\r)\"")
			if j < 0 || rest[1+j] != ')' {
				return "", "", 0, false
			}
			return s[open+1 : i], rest[1 : 1+j], i + 1 + 1 + j + 1, true
		}
	}
	return "", "", 0, false
}
