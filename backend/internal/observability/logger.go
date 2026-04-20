// Package observability provides structured logging for the application.
package observability

import (
	"log/slog"
	"os"
)

// NewLogger creates a structured JSON logger suitable for production (AKS + Log Analytics).
func NewLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}
