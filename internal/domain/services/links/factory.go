package links

import (
	"fmt"
	"sort"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	content_excluder "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/content_excluder"
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
)

// LinkFactory finds every link in a file's content: it first lets each
// registered ContentExcluder blank the text no link is searched in, then
// runs each registered LinkParser over the result, and makes sure no two
// parsers claim the same text. It is the entity.LinkSearcher that
// entity.File.Links calls through (via entity.SetDefaultLinkSearcher).
type LinkFactory struct {
	excluders []content_excluder.ContentExcluder
	parsers   []link_parser.LinkParser
}

var _ entity.LinkSearcher = (*LinkFactory)(nil)

// NewLinkFactory registers excluders and parsers, each run in the given
// order; every excluder masks the content the previous one returned.
func NewLinkFactory(excluders []content_excluder.ContentExcluder, parsers []link_parser.LinkParser) *LinkFactory {
	return &LinkFactory{excluders: excluders, parsers: parsers}
}

// NewDefaultLinkFactory registers every excluder and link syntax skills are
// written in.
func NewDefaultLinkFactory() *LinkFactory {
	return NewLinkFactory(
		[]content_excluder.ContentExcluder{content_excluder.ExampleFence{}, content_excluder.InlineCode{}},
		[]link_parser.LinkParser{link_parser.MarkdownParser{}, link_parser.WikilinkParser{}},
	)
}

// SearchLinks returns the links of all registered parsers ordered by
// Start, with Start, End and Raw taken from the span each parser found. A
// link touching excluded text (e.g. "[`x`](y)") is dropped.
// Fails with "invalid-link-span" if a parser reports a span that is empty
// or outside content, with the parser's own error if it cannot parse a span
// it found, and with "link-overlap" if two found spans share any text.
func (f *LinkFactory) SearchLinks(content string) ([]*model.Link, error) {
	out := []*model.Link{}
	searchable := content
	var excluded []link_parser.Span
	for _, e := range f.excluders {
		var spans []link_parser.Span
		searchable, spans = e.Mask(searchable)
		excluded = append(excluded, spans...)
	}
	for _, p := range f.parsers {
		for _, s := range p.Find(searchable) {
			if s.Start < 0 || s.End > len(content) || s.Start >= s.End {
				return nil, model.Issue{Code: "invalid-link-span", Message: fmt.Sprintf("span [%d,%d) outside content of length %d", s.Start, s.End, len(content))}
			}
			if intersects(s, excluded) {
				continue
			}
			raw := content[s.Start:s.End]
			l, err := p.Parse(raw)
			if err != nil {
				return nil, err
			}
			l.Start, l.End, l.Raw = s.Start, s.End, raw
			out = append(out, &l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	// Sorted by Start, any overlap shows up between neighbours.
	for i := 1; i < len(out); i++ {
		prev, next := out[i-1], out[i]
		if next.Start < prev.End {
			return nil, model.Issue{Code: "link-overlap", Link: next.Raw, Message: fmt.Sprintf("%s link %q at [%d,%d) overlaps %s link %q at [%d,%d)", next.Format, next.Raw, next.Start, next.End, prev.Format, prev.Raw, prev.Start, prev.End)}
		}
	}
	return out, nil
}

// intersects reports whether s shares any byte with one of spans.
func intersects(s link_parser.Span, spans []link_parser.Span) bool {
	for _, x := range spans {
		if s.Start < x.End && s.End > x.Start {
			return true
		}
	}
	return false
}
