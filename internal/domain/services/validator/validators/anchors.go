package validator

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	content_excluder "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/content_excluder"
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
)

var (
	atxHeading = regexp.MustCompile(`^ {0,3}#{1,6}(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$`)
	blockID    = regexp.MustCompile(`(?:^|\s)\^([A-Za-z0-9-]+)[ \t]*$`)
	htmlAnchor = regexp.MustCompile(`(?i)\b(?:id|name)\s*=\s*["']([^"']+)["']`)
)

// hasAnchor reports whether markdown content defines the anchor fragment
// (a link's Fragment, with or without its "#", URL-encoded or not) names:
//   - "^id" -- a block id, a line ending in " ^id" (Obsidian);
//   - otherwise a heading, by its GitHub slug (repeats get "-1", "-2", ...)
//     or by its text ignoring case (as wikilinks write it);
//   - or an HTML id/name attribute.
//
// Only "#" headings count; headings and block ids inside fenced code
// blocks don't.
//
// Примеры (content -- "## Getting Started!\n## Getting Started!\nText ^note\n<a id=\"top\"></a>"):
//   - "#getting-started"   -> true (GitHub slug)
//   - "#getting-started-1" -> true (slug второго такого же заголовка)
//   - "#Getting Started!"  -> true (текст заголовка, как в wikilink)
//   - "#getting%20started" -> false (декодируется в "getting started", а это
//     не slug и не текст заголовка)
//   - "#^note"             -> true (block id)
//   - "#top"               -> true (HTML id)
//   - "#missing"           -> false
func hasAnchor(content, fragment string) bool {
	want := strings.TrimPrefix(fragment, "#")
	if decoded, err := url.PathUnescape(want); err == nil {
		want = decoded
	}
	if want == "" {
		return true
	}
	fences := content_excluder.CodeFences(content)
	slugs := map[string]int{}
	offset := 0
	for _, line := range strings.SplitAfter(content, "\n") {
		start := offset
		offset += len(line)
		if inFence(start, fences) {
			continue
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(want, "^") {
			if m := blockID.FindStringSubmatch(line); m != nil && m[1] == want[1:] {
				return true
			}
			continue
		}
		if m := atxHeading.FindStringSubmatch(line); m != nil {
			text := strings.TrimSpace(m[1])
			slug := githubSlug(text)
			if n := slugs[slug]; n > 0 {
				slugs[slug] = n + 1
				slug = slug + "-" + strconv.Itoa(n)
			} else {
				slugs[slug] = 1
			}
			if strings.EqualFold(want, slug) || strings.EqualFold(want, text) {
				return true
			}
		}
		for _, m := range htmlAnchor.FindAllStringSubmatch(line, -1) {
			if m[1] == want {
				return true
			}
		}
	}
	return false
}

// githubSlug is the anchor GitHub gives a heading: lowercase, every
// character but letters, digits, spaces, "-" and "_" dropped, spaces
// turned into "-".
//
// Примеры:
//   - "Getting Started!" -> "getting-started"
//   - "`code` & more"    -> "code--more"
//   - "Шаг 2: Установка" -> "шаг-2-установка"
func githubSlug(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func inFence(offset int, fences []link_parser.Span) bool {
	for _, f := range fences {
		if offset >= f.Start && offset < f.End {
			return true
		}
	}
	return false
}
