// Package logging configures the structured logger used across the application.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog.Logger configured for the given env ("dev" or "prod").
// Dev uses text output; prod uses JSON.
func New(env string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if strings.EqualFold(env, "dev") {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
