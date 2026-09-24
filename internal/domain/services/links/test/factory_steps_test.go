package links_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/InsonusK/go-ai-skill-manage/internal/domain/model"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	content_excluder "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/content_excluder"
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

// fakeParser reports preset spans and parses any raw text into a Link whose
// Format is its name and whose Path is the raw text, leaving Start, End and
// Raw for the factory to fill. It records the content it was asked to search.
type fakeParser struct {
	name     string
	spans    []link_parser.Span
	fails    map[string]bool
	parsed   []string
	searched []string
}

func (p *fakeParser) Find(content string) []link_parser.Span {
	p.searched = append(p.searched, content)
	return p.spans
}

func (p *fakeParser) Parse(raw string) (model.Link, error) {
	p.parsed = append(p.parsed, raw)
	if p.fails[raw] {
		return model.Link{}, model.Issue{Code: "invalid-link", Link: raw, Message: "fake failure"}
	}
	return model.Link{Path: raw, Format: p.name}, nil
}

// fakeExcluder hides preset spans and records the content it was asked to
// search.
type fakeExcluder struct {
	spans    []link_parser.Span
	searched []string
}

func (e *fakeExcluder) Search(content string) []link_parser.Span {
	e.searched = append(e.searched, content)
	return e.spans
}

func (e *fakeExcluder) Mask(content string) (string, []link_parser.Span) {
	spans := e.Search(content)
	return content_excluder.MaskSpans(content, spans), spans
}

// parseSpans reads "start-end,start-end" into spans.
func parseSpans(spans string) ([]link_parser.Span, error) {
	var out []link_parser.Span
	for _, s := range strings.Split(spans, ",") {
		start, end, ok := strings.Cut(s, "-")
		if !ok {
			return nil, fmt.Errorf("span %q is not start-end", s)
		}
		a, err := strconv.Atoi(start)
		if err != nil {
			return nil, err
		}
		b, err := strconv.Atoi(end)
		if err != nil {
			return nil, err
		}
		out = append(out, link_parser.Span{Start: a, End: b})
	}
	return out, nil
}

func initializeFactory(sc *godog.ScenarioContext) {
	// factory is built on first search from the fakes registered so far,
	// unless a step set it already.
	var factory *links.LinkFactory
	var fakes map[string]*fakeParser
	var parsers []link_parser.LinkParser
	var excluderFakes map[string]*fakeExcluder
	var excluders []content_excluder.ContentExcluder
	var found []*model.Link
	var searchErr error
	sc.Step(`^fake parsers "([^"]*)" are registered$`, func(ctx context.Context, names string) error {
		fakes = map[string]*fakeParser{}
		for _, name := range strings.Split(names, ",") {
			if name == "" {
				continue
			}
			fakes[name] = &fakeParser{name: name, fails: map[string]bool{}}
			parsers = append(parsers, fakes[name])
		}
		testsupport.Log("parsers=%s", names)
		return nil
	})
	sc.Step(`^fake excluder "([^"]*)" hides spans "([^"]*)"$`, func(ctx context.Context, name, spans string) error {
		parsed, err := parseSpans(spans)
		if err != nil {
			return err
		}
		if excluderFakes == nil {
			excluderFakes = map[string]*fakeExcluder{}
		}
		excluderFakes[name] = &fakeExcluder{spans: parsed}
		excluders = append(excluders, excluderFakes[name])
		testsupport.Log("excluder=%s spans=%+v", name, parsed)
		return nil
	})
	sc.Step(`^fake excluder "([^"]*)" searched$`, func(ctx context.Context, name string, d *godog.DocString) error {
		return testsupport.JSON(excluderFakes[name].searched, d)
	})
	sc.Step(`^the default link factory$`, func(ctx context.Context) error {
		factory = links.NewDefaultLinkFactory()
		return nil
	})
	sc.Step(`^fake parser "([^"]*)" reports spans "([^"]*)"$`, func(ctx context.Context, name, spans string) error {
		parsed, err := parseSpans(spans)
		if err != nil {
			return err
		}
		fakes[name].spans = append(fakes[name].spans, parsed...)
		testsupport.Log("parser=%s spans=%+v", name, fakes[name].spans)
		return nil
	})
	sc.Step(`^fake parser "([^"]*)" fails to parse "([^"]*)"$`, func(ctx context.Context, name, raw string) error {
		fakes[name].fails[raw] = true
		return nil
	})
	sc.Step(`^I search links in$`, func(ctx context.Context, d *godog.DocString) error {
		if factory == nil {
			factory = links.NewLinkFactory(excluders, parsers)
		}
		found, searchErr = factory.SearchLinks(d.Content)
		testsupport.Log("content=%s error=%v", d.Content, searchErr)
		return nil
	})
	sc.Step(`^the searched links are$`, func(ctx context.Context, d *godog.DocString) error {
		if searchErr != nil {
			return searchErr
		}
		return testsupport.JSON(found, d)
	})
	sc.Step(`^searching fails with "([^"]*)"$`, func(ctx context.Context, code string) error {
		return issueCode(searchErr, code)
	})
	sc.Step(`^the search error mentions "([^"]*)"$`, func(ctx context.Context, s string) error {
		if searchErr == nil || !strings.Contains(searchErr.Error(), s) {
			return fmt.Errorf("error=%v; want it to mention %q", searchErr, s)
		}
		return nil
	})
	sc.Step(`^the searched link raws are$`, func(ctx context.Context, d *godog.DocString) error {
		if searchErr != nil {
			return searchErr
		}
		raws := []string{}
		for _, l := range found {
			raws = append(raws, l.Raw)
		}
		return testsupport.JSON(raws, d)
	})
	sc.Step(`^fake parser "([^"]*)" searched$`, func(ctx context.Context, name string, d *godog.DocString) error {
		return testsupport.JSON(fakes[name].searched, d)
	})
	sc.Step(`^fake parser "([^"]*)" parsed$`, func(ctx context.Context, name string, d *godog.DocString) error {
		return testsupport.JSON(fakes[name].parsed, d)
	})
}

// issueCode checks that err is a model.Issue carrying the given code.
func issueCode(err error, code string) error {
	var issue model.Issue
	if !errors.As(err, &issue) {
		return fmt.Errorf("error=%v; want Issue %s", err, code)
	}
	return testsupport.Equal(issue.Code, code)
}
