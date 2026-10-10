package logging_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/InsonusK/go-ai-skill-manager/internal/logging"
	"github.com/InsonusK/go-ai-skill-manager/tools/testsupport"
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
		logger := logging.Init(&output, v == "true", false)
		logger.Debug("debug-event")
		logger.Info("info-event")
		testsupport.Log("log=%s", output.String())
		return nil
	})
	sc.Step(`^log contains debug "([^"]*)" and info "([^"]*)"$`, func(ctx context.Context, d, i string) error {
		return testsupport.Equal([]bool{strings.Contains(output.String(), "debug-event"), strings.Contains(output.String(), "info-event")}, []bool{d == "true", i == "true"})
	})
	sc.Step(`^I log at level "([A-Z]+)" with color "([^"]*)"$`, func(ctx context.Context, level, color string) error {
		output.Reset()
		previous := slog.Default()
		defer slog.SetDefault(previous)
		logger := logging.Init(&output, true, color == "true")
		levels := map[string]slog.Level{
			"DEBUG": slog.LevelDebug,
			"INFO":  slog.LevelInfo,
			"WARN":  slog.LevelWarn,
			"ERROR": slog.LevelError,
		}
		logger.Log(ctx, levels[level], "event")
		testsupport.Log("level=%s color=%s log=%q", level, color, output.String())
		return nil
	})
	sc.Step(`^log level "([A-Z]+)" has ANSI color "([0-9]+)"$`, func(ctx context.Context, level, code string) error {
		want := "\x1b[" + code + "mlevel=" + level + "\x1b[0m"
		if !strings.Contains(output.String(), want) {
			return fmt.Errorf("log lacks %q: %q", want, output.String())
		}
		return nil
	})
	sc.Step(`^log contains ANSI "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(strings.Contains(output.String(), "\x1b["), want == "true")
	})
	var colorEnabled bool
	sc.Step(`^color mode is "([^"]*)", output terminal is "([^"]*)" and NO_COLOR is "([^"]*)"$`, func(ctx context.Context, mode, terminal, noColor string) error {
		value := ""
		if noColor == "true" {
			value = "1"
		}
		colorEnabled = logging.ColorEnabled(mode, terminal == "true", value)
		testsupport.Log("mode=%s terminal=%s no_color=%s enabled=%t", mode, terminal, noColor, colorEnabled)
		return nil
	})
	sc.Step(`^color is enabled "([^"]*)"$`, func(ctx context.Context, want string) error {
		return testsupport.Equal(colorEnabled, want == "true")
	})
}
