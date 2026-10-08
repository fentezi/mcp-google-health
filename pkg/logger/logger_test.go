package logger_test

import (
	"log/slog"
	"testing"

	"github.com/fentezi/mcp-google-health/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestNew_Level(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level string
		want  slog.Level
	}{
		{level: "debug", want: slog.LevelDebug},
		{level: "info", want: slog.LevelInfo},
		{level: "warn", want: slog.LevelWarn},
		{level: "error", want: slog.LevelError},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()

			log := logger.New(tt.level)

			assert.True(t, log.Enabled(t.Context(), tt.want), "level %s must be enabled", tt.want)
			assert.False(t, log.Enabled(t.Context(), tt.want-1), "level below %s must be disabled", tt.want)
		})
	}
}
