package content_excluder

import (
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
)

// ContentExcluder knows one kind of text in a file's content that holds no
// links to follow -- code a skill shows, not links it makes -- and hides it
// from link parsers. LinkFactory drives a set of them.
type ContentExcluder interface {
	// Search returns the span of every part of content this excluder hides.
	Search(content string) []link_parser.Span
	// Mask calls Search and returns content with every found span blanked
	// (see MaskSpans), together with those spans.
	Mask(content string) (string, []link_parser.Span)
}

// MaskSpans blanks every span of content with spaces, keeping newlines, so
// the result has the same length and every other byte keeps its offset.
func MaskSpans(content string, spans []link_parser.Span) string {
	if len(spans) == 0 {
		return content
	}
	b := []byte(content)
	for _, s := range spans {
		for i := s.Start; i < s.End; i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}
	return string(b)
}
