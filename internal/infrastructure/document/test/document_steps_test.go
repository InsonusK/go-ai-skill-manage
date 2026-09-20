package document_test

import (
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manage/internal/infrastructure/document"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var text string
	var actual map[string]any
	var failure error
	sc.Step(`^a document$`, func(ctx context.Context, d *godog.DocString) error {
		text = d.Content
		failure = nil
		testsupport.Log("input=%s", text)
		return nil
	})
	sc.Step(`^I round trip the document$`, func(ctx context.Context) error {
		codec := document.Codec{}
		d, err := codec.Decode([]byte(text))
		failure = err
		if err != nil {
			return nil
		}
		raw, err := codec.Encode(d)
		if err != nil {
			return err
		}
		round, err := codec.Decode(raw)
		if err != nil {
			return err
		}
		actual = round.Properties
		if actual == nil {
			actual = map[string]any{}
		}
		actual["body"] = round.Body
		testsupport.Log("round trip=%s", raw)
		return nil
	})
	sc.Step(`^the document result is$`, func(ctx context.Context, d *godog.DocString) error {
		if failure != nil {
			return failure
		}
		return testsupport.JSON(actual, d)
	})
	sc.Step(`^the document error contains "([^"]*)"$`, func(ctx context.Context, s string) error {
		testsupport.Log("error=%v", failure)
		if failure == nil || !strings.Contains(failure.Error(), s) {
			return fmt.Errorf("error=%v; want %s", failure, s)
		}
		return nil
	})
}
