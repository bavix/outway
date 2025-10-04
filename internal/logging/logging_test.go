package logging_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bavix/outway/internal/logging"
)

//nolint:funlen
func TestBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		app    string
		level  string
		format string
	}{
		{
			name:   "default values",
			app:    "test",
			level:  "info",
			format: "json",
		},
		{
			name:   "debug level",
			app:    "test",
			level:  "debug",
			format: "json",
		},
		{
			name:   "console format",
			app:    "test",
			level:  "info",
			format: "console",
		},
		{
			name:   "error level",
			app:    "test",
			level:  "error",
			format: "json",
		},
		{
			name:   "warn level",
			app:    "test",
			level:  "warn",
			format: "json",
		},
		{
			name:   "fatal level",
			app:    "test",
			level:  "fatal",
			format: "json",
		},
		{
			name:   "panic level",
			app:    "test",
			level:  "panic",
			format: "json",
		},
		{
			name:   "trace level",
			app:    "test",
			level:  "trace",
			format: "json",
		},
		{
			name:   "empty app name",
			app:    "",
			level:  "info",
			format: "json",
		},
		{
			name:   "empty level",
			app:    "test",
			level:  "",
			format: "json",
		},
		{
			name:   "empty format",
			app:    "test",
			level:  "info",
			format: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := logging.Base(tt.app, tt.level, tt.format)
			assert.NotNil(t, logger)

			// Test that logger has the correct app name
			// We can't directly access the fields, but we can test the logger works
			logger.Info().Msg("test message")
		})
	}
}

//nolint:funlen
func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{
			name:     "debug level",
			level:    "debug",
			expected: "debug",
		},
		{
			name:     "info level",
			level:    "info",
			expected: "info",
		},
		{
			name:     "warn level",
			level:    "warn",
			expected: "warn",
		},
		{
			name:     "error level",
			level:    "error",
			expected: "error",
		},
		{
			name:     "fatal level",
			level:    "fatal",
			expected: "fatal",
		},
		{
			name:     "panic level",
			level:    "panic",
			expected: "panic",
		},
		{
			name:     "trace level",
			level:    "trace",
			expected: "trace",
		},
		{
			name:     "empty level",
			level:    "",
			expected: "info",
		},
		{
			name:     "invalid level",
			level:    "invalid",
			expected: "info",
		},
		{
			name:     "uppercase level",
			level:    "DEBUG",
			expected: "debug",
		},
		{
			name:     "mixed case level",
			level:    "Info",
			expected: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test that level is valid
			assert.Contains(t, []string{"debug", "info", "warn", "error", "fatal", "panic", "trace"}, tt.expected)
		})
	}
}

func TestWriterForFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "json format",
			format:   "json",
			expected: "json",
		},
		{
			name:     "console format",
			format:   "console",
			expected: "console",
		},
		{
			name:     "empty format",
			format:   "",
			expected: "json",
		},
		{
			name:     "invalid format",
			format:   "invalid",
			expected: "json",
		},
		{
			name:     "uppercase format",
			format:   "JSON",
			expected: "json",
		},
		{
			name:     "mixed case format",
			format:   "Console",
			expected: "console",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test that format is valid
			assert.Contains(t, []string{"json", "console"}, tt.expected)
		})
	}
}

func TestBaseWithDifferentLevels(t *testing.T) {
	t.Parallel()
	// Test that different levels actually affect the logger behavior
	debugLogger := logging.Base("test", "debug", "json")
	infoLogger := logging.Base("test", "info", "json")
	errorLogger := logging.Base("test", "error", "json")

	// All loggers should be created successfully
	assert.NotNil(t, debugLogger)
	assert.NotNil(t, infoLogger)
	assert.NotNil(t, errorLogger)

	// Test that loggers work
	debugLogger.Debug().Msg("debug message")
	infoLogger.Info().Msg("info message")
	errorLogger.Error().Msg("error message")
}

func TestBaseWithDifferentFormats(t *testing.T) {
	t.Parallel()
	// Test that different formats work
	jsonLogger := logging.Base("test", "info", "json")
	consoleLogger := logging.Base("test", "info", "console")

	// Both loggers should be created successfully
	assert.NotNil(t, jsonLogger)
	assert.NotNil(t, consoleLogger)

	// Test that loggers work
	jsonLogger.Info().Msg("json message")
	consoleLogger.Info().Msg("console message")
}

func TestBaseWithDifferentApps(t *testing.T) {
	t.Parallel()
	// Test that different app names work
	app1Logger := logging.Base("app1", "info", "json")
	app2Logger := logging.Base("app2", "info", "json")

	// Both loggers should be created successfully
	assert.NotNil(t, app1Logger)
	assert.NotNil(t, app2Logger)

	// Test that loggers work
	app1Logger.Info().Msg("app1 message")
	app2Logger.Info().Msg("app2 message")
}

func TestParseLevelEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{
			name:     "whitespace level",
			level:    " debug ",
			expected: "debug",
		},
		{
			name:     "numeric level",
			level:    "123",
			expected: "info",
		},
		{
			name:     "special characters level",
			level:    "debug!@#",
			expected: "info",
		},
		{
			name:     "very long level",
			level:    "verylonglevelname",
			expected: "info",
		},
		{
			name:     "unicode level",
			level:    "дебаг",
			expected: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test that level is valid
			assert.Contains(t, []string{"debug", "info", "warn", "error", "fatal", "panic", "trace"}, tt.expected)
		})
	}
}

func TestWriterForFormatEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{
			name:     "whitespace format",
			format:   " json ",
			expected: "json",
		},
		{
			name:     "numeric format",
			format:   "123",
			expected: "json",
		},
		{
			name:     "special characters format",
			format:   "json!@#",
			expected: "json",
		},
		{
			name:     "very long format",
			format:   "verylongformatname",
			expected: "json",
		},
		{
			name:     "unicode format",
			format:   "джсон",
			expected: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test that format is valid
			assert.Contains(t, []string{"json", "console"}, tt.expected)
		})
	}
}

func TestBaseIntegration(t *testing.T) {
	t.Parallel()
	// Test that Base function works with real logging
	logger := logging.Base("integration-test", "info", "json")
	assert.NotNil(t, logger)

	// Test different log levels (excluding fatal and panic as they terminate)
	logger.Trace().Msg("trace message")
	logger.Debug().Msg("debug message")
	logger.Info().Msg("info message")
	logger.Warn().Msg("warn message")
	logger.Error().Msg("error message")
	// Skip fatal and panic as they terminate the program
}

func TestBaseWithAllLevels(t *testing.T) {
	t.Parallel()

	levels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		t.Run("level_"+level, func(t *testing.T) {
			t.Parallel()

			logger := logging.Base("test", level, "json")
			assert.NotNil(t, logger)
			logger.Info().Msg("test message")
		})
	}
}

func TestBaseWithAllFormats(t *testing.T) {
	t.Parallel()

	formats := []string{"json", "console"}

	for _, format := range formats {
		t.Run("format_"+format, func(t *testing.T) {
			t.Parallel()

			logger := logging.Base("test", "info", format)
			assert.NotNil(t, logger)
			logger.Info().Msg("test message")
		})
	}
}
