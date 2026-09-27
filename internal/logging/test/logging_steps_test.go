package logging_test

import (
	"bytes"
	"context"
	"github.com/InsonusK/go-ai-skill-manage/internal/logging"
	"github.com/InsonusK/go-ai-skill-manage/tools/testsupport"
	"github.com/cucumber/godog"
	"log/slog"
	"strings"
)

func initialize(sc *godog.ScenarioContext) {
	var output bytes.Buffer
	sc.Step(`^I log with debug "([^"]*)"$`, func(ctx context.Context, v string) error {
		output.Reset()
		previous := slog.Default()
		defer slog.SetDefault(previous)
		logger := logging.Init(&output, v == "true")
		logger.Debug("debug-event")
		logger.Info("info-event")
		testsupport.Log("log=%s", output.String())
		return nil
	})
	sc.Step(`^log contains debug "([^"]*)" and info "([^"]*)"$`, func(ctx context.Context, d, i string) error {
		return testsupport.Equal([]bool{strings.Contains(output.String(), "debug-event"), strings.Contains(output.String(), "info-event")}, []bool{d == "true", i == "true"})
	})
}
