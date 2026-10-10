package links

import (
	"fmt"
	"sort"

	"github.com/InsonusK/go-ai-skill-manager/internal/domain/entity"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manager/internal/domain/model/issues"
	content_excluder "github.com/InsonusK/go-ai-skill-manager/internal/domain/services/links/content_excluder"
	link_parser "github.com/InsonusK/go-ai-skill-manager/internal/domain/services/links/parser"
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
// Start, with Start and End taken from the span each parser found.
// Excluded text inside a link's label is part of the label: "[`x`](y)" and
// "[[y|`x`]]" are links to y. A link touching excluded text anywhere else
// -- it lies in the excluded text, crosses its border, or has it in its
// target -- is dropped.
// Fails with "invalid-link-span" if a parser reports a span that is empty
// or outside content, with the parser's own error if it cannot parse a span
// it found, and with "link-overlap" if two found spans share any text.
func (f *LinkFactory) SearchLinks(content string) ([]model.ParsedLink, error) {
	out := []model.ParsedLink{}
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
				return nil, issues.SkillIssue{Code: issues.CodeInvalidLinkSpan, Message: fmt.Sprintf("span [%d,%d) outside content of length %d", s.Start, s.End, len(content))}
			}
			touched := intersecting(s, excluded)
			if !within(s, touched) {
				continue
			}
			l, err := p.Parse(content[s.Start:s.End])
			if len(touched) > 0 {
				// The masked link has the same target only when no
				// excluded text is part of it. A link that is one only
				// while masked (a "]" inside the code of its label) is
				// dropped too.
				masked, maskedErr := p.Parse(searchable[s.Start:s.End])
				if err != nil || maskedErr != nil || masked.Path != l.Path || masked.Fragment != l.Fragment {
					continue
				}
			}
			if err != nil {
				return nil, err
			}
			l.Start, l.End = s.Start, s.End
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	// Sorted by Start, any overlap shows up between neighbours.
	for i := 1; i < len(out); i++ {
		prev, next := out[i-1], out[i]
		if next.Start < prev.End {
			nextRaw, prevRaw := content[next.Start:next.End], content[prev.Start:prev.End]
			return nil, issues.SkillIssue{Code: issues.CodeLinkOverlap, Link: nextRaw, Message: fmt.Sprintf("%s link %q at [%d,%d) overlaps %s link %q at [%d,%d)", next.Format, nextRaw, next.Start, next.End, prev.Format, prevRaw, prev.Start, prev.End)}
		}
	}
	return out, nil
}

// intersecting returns the spans that share any byte with s.
func intersecting(s link_parser.Span, spans []link_parser.Span) []link_parser.Span {
	var out []link_parser.Span
	for _, x := range spans {
		if s.Start < x.End && s.End > x.Start {
			out = append(out, x)
		}
	}
	return out
}

// within reports whether every one of spans lies strictly inside s.
func within(s link_parser.Span, spans []link_parser.Span) bool {
	for _, x := range spans {
		if x.Start <= s.Start || x.End >= s.End {
			return false
		}
	}
	return true
}
