package content_excluder_test

import (
	"context"

	content_excluder "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/content_excluder"
	link_parser "github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links/parser"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
)

var excluders = map[string]content_excluder.ContentExcluder{
	"example fences": content_excluder.ExampleFence{},
	"inline code":    content_excluder.InlineCode{},
}

func initialize(sc *godog.ScenarioContext) {
	var spans []link_parser.Span
	var masked string
	sc.Step(`^I search (example fences|inline code) in$`, func(ctx context.Context, kind string, d *godog.DocString) error {
		spans = excluders[kind].Search(d.Content)
		testsupport.Log("excluder=%s content=%q spans=%+v", kind, d.Content, spans)
		return nil
	})
	sc.Step(`^I mask (example fences|inline code) in$`, func(ctx context.Context, kind string, d *godog.DocString) error {
		masked, spans = excluders[kind].Mask(d.Content)
		testsupport.Log("excluder=%s content=%q masked=%q spans=%+v", kind, d.Content, masked, spans)
		return nil
	})
	sc.Step(`^the excluded spans are$`, func(ctx context.Context, d *godog.DocString) error {
		actual := [][2]int{}
		for _, s := range spans {
			actual = append(actual, [2]int{s.Start, s.End})
		}
		return testsupport.JSON(actual, d)
	})
	sc.Step(`^the masked content is$`, func(ctx context.Context, d *godog.DocString) error {
		return testsupport.JSON(masked, d)
	})
}
