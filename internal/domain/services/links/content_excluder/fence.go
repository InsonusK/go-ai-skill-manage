package content_excluder

import (
	"strings"

	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
)

type fence struct {
	start, end int
	example    bool
}

// fenced returns every fenced code block of text (``` or ~~~, indented at
// most 3 spaces), each ending after its closing fence line or at the end of
// text when unclosed; example is set when the info string is "example".
func fenced(text string) []fence {
	var out []fence
	offset := 0
	start := -1
	marker := ""
	example := false
	for _, line := range strings.SplitAfter(text, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)
		if indent <= 3 {
			token := strings.TrimSpace(trimmed)
			if start < 0 && (strings.HasPrefix(token, "```") || strings.HasPrefix(token, "~~~")) {
				char := token[0]
				n := 0
				for n < len(token) && token[n] == char {
					n++
				}
				marker = token[:n]
				start = offset
				example = strings.TrimSpace(token[n:]) == "example"
			} else if start >= 0 && strings.HasPrefix(token, marker) && strings.Trim(token, string(marker[0])) == "" {
				out = append(out, fence{start, offset + len(line), example})
				start = -1
			}
		}
		offset += len(line)
	}
	if start >= 0 {
		out = append(out, fence{start, len(text), example})
	}
	return out
}

// CodeFences returns the span of every fenced code block of content (see
// fenced), whatever its info string -- fence lines included.
func CodeFences(content string) []link_parser.Span {
	out := []link_parser.Span{}
	for _, f := range fenced(content) {
		out = append(out, link_parser.Span{Start: f.start, End: f.end})
	}
	return out
}
