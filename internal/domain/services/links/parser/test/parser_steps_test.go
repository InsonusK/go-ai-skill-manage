package link_parser_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

func initialize(sc *godog.ScenarioContext) {
	var raw, format string
	var parsed model.Link
	var parseErr error
	var spans []link_parser.Span
	parse := func(p link_parser.LinkParser, f, r string) {
		raw, format = r, f
		parsed, parseErr = p.Parse(r)
		testsupport.Log("raw=%s link=%+v error=%v", r, parsed, parseErr)
	}
	find := func(p link_parser.LinkParser, content string) {
		spans = p.Find(content)
		testsupport.Log("content=%s spans=%+v", content, spans)
	}
	sc.Step(`^I parse the markdown link (.+)$`, func(ctx context.Context, r string) error {
		parse(link_parser.MarkdownParser{}, "markdown", r)
		return nil
	})
	sc.Step(`^I parse the wikilink (.+)$`, func(ctx context.Context, r string) error {
		parse(link_parser.WikilinkParser{}, "wikilink", r)
		return nil
	})
	sc.Step(`^I find markdown links in$`, func(ctx context.Context, d *godog.DocString) error {
		find(link_parser.MarkdownParser{}, d.Content)
		return nil
	})
	sc.Step(`^I find wikilinks in$`, func(ctx context.Context, d *godog.DocString) error {
		find(link_parser.WikilinkParser{}, d.Content)
		return nil
	})
	sc.Step(`^the parsed link has text "([^"]*)" path "([^"]*)" fragment "([^"]*)" image (true|false) external (true|false)$`, func(ctx context.Context, text, p, fragment, image, external string) error {
		if parseErr != nil {
			return parseErr
		}
		return testsupport.Equal(parsed, model.Link{Raw: raw, Text: text, Path: p, Fragment: fragment, Format: format, Image: image == "true", External: external == "true"})
	})
	sc.Step(`^parsing fails with "([^"]*)"$`, func(ctx context.Context, code string) error {
		var issue model.Issue
		if !errors.As(parseErr, &issue) {
			return fmt.Errorf("error=%v; want Issue %s", parseErr, code)
		}
		return testsupport.Equal(issue.Code, code)
	})
	sc.Step(`^the found spans are$`, func(ctx context.Context, d *godog.DocString) error {
		actual := [][2]int{}
		for _, s := range spans {
			actual = append(actual, [2]int{s.Start, s.End})
		}
		return testsupport.JSON(actual, d)
	})
}
