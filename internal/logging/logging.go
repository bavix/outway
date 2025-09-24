package logging

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// Base builds a zerolog.Logger with level/format applied per-call.
// format: json|console; level: debug|info|warn|error
func Base(app, level, format string) zerolog.Logger {
	lvl := parseLevel(level)
	w := writerForFormat(format)

	return zerolog.New(w).Level(lvl).With().Timestamp().Str("app", app).Logger()
}

func parseLevel(s string) zerolog.Level {
	if lvl, err := zerolog.ParseLevel(strings.ToLower(strings.TrimSpace(s))); err == nil {
		return lvl
	}
	return zerolog.InfoLevel
}

func writerForFormat(format string) io.Writer {
	if strings.ToLower(format) == "console" {
		return zerolog.ConsoleWriter{Out: os.Stdout}
	}

	return os.Stdout
}
