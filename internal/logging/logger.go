// Package logging configures process-wide structured logging.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

func Init(out io.Writer, debug, color bool) *slog.Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	if color {
		out = levelColorWriter{out: out}
	}
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return logger
}

// ColorEnabled resolves a validated CLI color mode for a particular output.
func ColorEnabled(mode string, terminal bool, noColor string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	case "auto":
		return terminal && noColor == ""
	default:
		panic("invalid color mode: " + mode)
	}
}

// levelColorWriter decorates the stable level field produced by
// slog.TextHandler. It reports the size of the original write because ANSI
// bytes are presentation details added after the handler serialized a record.
type levelColorWriter struct{ out io.Writer }

func (w levelColorWriter) Write(p []byte) (int, error) {
	line := string(p)
	for level, code := range map[string]string{
		"DEBUG": "90",
		"INFO":  "36",
		"WARN":  "33",
		"ERROR": "31",
	} {
		plain := "level=" + level
		colored := "\x1b[" + code + "m" + plain + "\x1b[0m"
		line = strings.Replace(line, plain, colored, 1)
	}
	if _, err := io.WriteString(w.out, line); err != nil {
		return 0, err
	}
	return len(p), nil
}
