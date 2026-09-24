package link_parser

import (
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

// Span is the [Start, End) byte range of one link inside a file's content.
type Span struct{ Start, End int }

// LinkParser knows one link syntax (markdown, wikilink, ...): it finds the
// links of that syntax in a file's content and parses one raw link of that
// syntax into a model.ParsedLink. LinkFactory drives a set of them.
type LinkParser interface {
	// Find returns the span of every link of this parser's syntax in content.
	Find(content string) []Span
	// Parse turns one raw link, exactly as it appears in the file, into a
	// ParsedLink. Start and End are left zero: a raw link carries no
	// position.
	// Fails with an "invalid-link" Issue if raw is not exactly one link of
	// this parser's syntax.
	Parse(raw string) (model.ParsedLink, error)
}

func invalidLink(raw, format string) error {
	return model.Issue{Code: "invalid-link", Link: raw, Message: "not a " + format + " link"}
}

func findSpans(re interface{ FindAllStringIndex(string, int) [][]int }, content string) []Span {
	out := []Span{}
	for _, m := range re.FindAllStringIndex(content, -1) {
		out = append(out, Span{Start: m[0], End: m[1]})
	}
	return out
}

// newLink builds the ParsedLink every parser returns from one raw link,
// its label and its "path#fragment" target: the fragment keeps its leading
// "#", and a raw starting with "!" is an Image.
func newLink(raw, label, target, format string) model.ParsedLink {
	p, fragment, ok := strings.Cut(target, "#")
	if ok {
		fragment = "#" + fragment
	}
	return model.ParsedLink{Text: label, Path: p, Fragment: fragment, Format: format, Image: strings.HasPrefix(raw, "!")}
}
