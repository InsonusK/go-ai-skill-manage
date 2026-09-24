package transform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/entity"
)

func Rewrite(content string, links []*entity.Link, destinations map[string]string) (string, error) {
	ordered := append([]*entity.Link{}, links...)
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
