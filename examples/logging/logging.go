package logging

import (
	"context"
	"io"
	"log/slog"
)

func NewJSONLogger(output io.Writer, service, environment string) *slog.Logger {
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	})
	return slog.New(handler).With("service", service, "environment", environment)
}

func RequestCompleted(ctx context.Context, logger *slog.Logger, route string, status int) {
	logger.InfoContext(ctx, "запрос завершён",
		"event", "http.request.completed",
		"route", route,
		"status", status,
	)
}
