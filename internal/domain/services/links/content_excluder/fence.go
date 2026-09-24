package content_excluder

import "strings"

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
