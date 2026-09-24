package content_excluder

import (
	"strings"

	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
)

// InlineCode hides inline code spans: text between a run of backticks and
// the next run of the same length, backticks included -- so a span opened
// by two backticks may hold a single one. An unmatched run hides nothing.
// Backticks inside any fenced block are not inline code (so a fence line is
// never taken for one).
type InlineCode struct{}

var _ ContentExcluder = InlineCode{}

func (InlineCode) Search(content string) []link_parser.Span {
	fences := fenced(content)
	out := []link_parser.Span{}
	for i := 0; i < len(content); {
		if content[i] != '`' {
			i++
			continue
		}
		inside := false
		for _, f := range fences {
			if i >= f.start && i < f.end {
				i = f.end
				inside = true
				break
			}
		}
		if inside {
			continue
		}
		start := i
		for i < len(content) && content[i] == '`' {
			i++
		}
		run := content[start:i]
		end := -1
		for j := i; j < len(content); {
			k := strings.Index(content[j:], run)
			if k < 0 {
				break
			}
			k += j
			e := k + len(run)
			if (k == 0 || content[k-1] != '`') && (e == len(content) || content[e] != '`') {
				end = e
				break
			}
			j = e
		}
		if end >= 0 {
			out = append(out, link_parser.Span{Start: start, End: end})
			i = end
		}
	}
	return out
}

func (c InlineCode) Mask(content string) (string, []link_parser.Span) {
	spans := c.Search(content)
	return MaskSpans(content, spans), spans
}
