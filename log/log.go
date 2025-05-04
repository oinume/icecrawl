package log

import (
	"context"
	"io"
	"log/slog"
)

type contextKey struct{}

func New(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, nil))
}

func FromContext(ctx context.Context) *slog.Logger {
	return ctx.Value(contextKey{}).(*slog.Logger)
}

func FC(ctx context.Context) *slog.Logger {
	return FromContext(ctx)
}

func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

func ErrorAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
