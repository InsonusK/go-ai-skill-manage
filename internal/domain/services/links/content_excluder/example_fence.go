package content_excluder

import (
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
)

// ExampleFence hides fenced code blocks tagged "example" ("```example" or
// "~~~example"), fence lines included, up to the end of content when the
// block is unclosed. Untagged and other-tagged blocks are not hidden
// (Python compatibility).
type ExampleFence struct{}

var _ ContentExcluder = ExampleFence{}

func (ExampleFence) Search(content string) []link_parser.Span {
	out := []link_parser.Span{}
	for _, f := range fenced(content) {
		if f.example {
			out = append(out, link_parser.Span{Start: f.start, End: f.end})
		}
	}
	return out
}

func (e ExampleFence) Mask(content string) (string, []link_parser.Span) {
	spans := e.Search(content)
	return MaskSpans(content, spans), spans
}
