package link_parser

import (
	"path"
	"regexp"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
)

const wikilinkPattern = `!?\[\[([^\]]+)\]\]`

var wikilinkFind = regexp.MustCompile(wikilinkPattern)
var wikilinkExact = regexp.MustCompile(`^` + wikilinkPattern + `$`)

// WikilinkParser handles "[[path#fragment|label]]" links and their
// "![[path|label]]" image form. The last "|" separates the label; a "\|"
// (a pipe escaped inside a markdown table) is accepted as that separator.
// Without a label, the label is the base name of the path.
type WikilinkParser struct{}

var _ LinkParser = WikilinkParser{}

func (WikilinkParser) Find(content string) []Span { return findSpans(wikilinkFind, content) }

func (WikilinkParser) Parse(raw string) (model.ParsedLink, error) {
	m := wikilinkExact.FindStringSubmatch(raw)
	if m == nil {
		return model.ParsedLink{}, invalidLink(raw, "wikilink")
	}
	target, label := m[1], ""
	pipe := strings.LastIndex(target, "|")
	if pipe >= 0 {
		target, label = strings.TrimSuffix(target[:pipe], "\\"), target[pipe+1:]
	}
	l := newLink(raw, label, target, "wikilink")
	if pipe < 0 && l.Path != "" {
		l.Text = path.Base(l.Path)
	}
	return l, nil
}
