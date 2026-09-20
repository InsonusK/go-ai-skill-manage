package transform_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/links"
	"github.com/InsonusK/go-ai-skill-manage/internal/domain/services/transform"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var input, output string
	var props map[string]any
	sc.Step(`^transformation document$`, func(ctx context.Context, d *godog.DocString) error {
		input = d.Content
		testsupport.Log("input=%s", input)
		return nil
	})
	sc.Step(`^I rewrite links to "([^"]*)"$`, func(ctx context.Context, p string) error {
		found := links.Extract(input)
		dest := map[string]string{}
		for i := range found {
			found[i].Target = found[i].Path
			dest[found[i].Target] = p
		}
		var err error
		output, err = transform.Rewrite(input, found, dest)
		return err
	})
	sc.Step(`^I transform Claude properties$`, func(ctx context.Context) error {
		codec := document.Codec{}
		d, err := codec.Decode([]byte(input))
		if err != nil {
			return err
		}
		transformed := transform.Claude(d)
		props = transformed.Properties
		raw, err := codec.Encode(transformed)
		output = string(raw)
		return err
	})
	sc.Step(`^transformed text is$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.Equal(output, d.Content) })
	sc.Step(`^transformed properties are$`, func(ctx context.Context, d *godog.DocString) error { return testsupport.JSON(props, d) })
	sc.Step(`^transformed text contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("output=%s", output)
		if !strings.Contains(output, s) {
			return fmt.Errorf("missing %q", s)
		}
		return nil
	})
}
