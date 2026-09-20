package links

import (
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"path"
	"regexp"
	"sort"
	"strings"
)

var markdown = regexp.MustCompile(`!?\[([^\]]*)\]\(([^\s\)"]*)\)`)
var wiki = regexp.MustCompile(`!?\[\[([^\]]+)\]\]`)

func Extract(content string) []model.Link {
	out := []model.Link{}
	for _, m := range markdown.FindAllStringSubmatchIndex(content, -1) {
		out = append(out, makeLink(content, m[0], m[1], content[m[2]:m[3]], content[m[4]:m[5]], "markdown"))
	}
	for _, m := range wiki.FindAllStringSubmatchIndex(content, -1) {
		inner := content[m[2]:m[3]]
		target := inner
		label := ""
		if i := strings.LastIndex(inner, "|"); i >= 0 {
			target = strings.TrimSuffix(inner[:i], "\\")
			label = inner[i+1:]
		} else {
			base, _, _ := strings.Cut(target, "#")
			label = path.Base(base)
			if base == "" {
				label = ""
			}
		}
		out = append(out, makeLink(content, m[0], m[1], label, target, "wikilink"))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}
func makeLink(content string, start, end int, label, target, format string) model.Link {
	p, fragment, ok := strings.Cut(target, "#")
	if ok {
		fragment = "#" + fragment
	}
	return model.Link{Start: start, End: end, Raw: content[start:end], Text: label, Path: p, Fragment: fragment, Format: format, Image: content[start] == '!'}
}
