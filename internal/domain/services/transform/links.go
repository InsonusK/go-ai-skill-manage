package transform

import (
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"sort"
	"strings"
)

func Rewrite(content string, links []model.Link, destinations map[string]string) (string, error) {
	ordered := append([]model.Link{}, links...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Start < ordered[j].Start })
	var b strings.Builder
	cursor := 0
	for _, l := range ordered {
		dest, ok := destinations[l.Target]
		if !ok {
			return "", fmt.Errorf("unresolved output target %q", l.Target)
		}
		if l.Start < cursor || l.End > len(content) || l.End < l.Start {
			return "", fmt.Errorf("invalid link span")
		}
		b.WriteString(content[cursor:l.Start])
		if l.Image {
			b.WriteByte('!')
		}
		b.WriteString("[" + l.Text + "](" + dest + l.Fragment + ")")
		cursor = l.End
	}
	b.WriteString(content[cursor:])
	return b.String(), nil
}
