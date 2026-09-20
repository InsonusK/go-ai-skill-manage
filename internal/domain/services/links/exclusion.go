package links

import (
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"strings"
)

type span struct {
	start, end int
	example    bool
}

func Excluded(content, file string, l model.Link, skip []string) bool {
	for _, segment := range strings.Split(strings.ReplaceAll(file, "\\", "/"), "/") {
		for _, folder := range skip {
			if segment == folder {
				return true
			}
		}
	}
	if l.Path == "" {
		return true
	}
	lower := strings.ToLower(l.Path)
	for _, prefix := range []string{"http://", "https://", "mailto:", "ftp://", "file://"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	fences := fenced(content)
	for _, f := range fences {
		if l.Start >= f.start && l.End <= f.end {
			return f.example
		}
	}
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
			if l.Start < end && l.End > start {
				return true
			}
			i = end
		}
	}
	return false
}
func fenced(text string) []span {
	var out []span
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
				out = append(out, span{start, offset + len(line), example})
				start = -1
			}
		}
		offset += len(line)
	}
	if start >= 0 {
		out = append(out, span{start, len(text), example})
	}
	return out
}
