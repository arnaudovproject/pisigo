// Pisigo framework — https://pisigo.com
// Author: Ventsislav Arnaudov — https://varnaudov.com

package pisigo

import (
	"context"
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

type slogLogger struct {
	l *slog.Logger
}

func NewLogger(level slog.Level) Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return &slogLogger{l: slog.New(handler)}
}

func NewTextLogger(level slog.Level) Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return &slogLogger{l: slog.New(handler)}
}

func DefaultLogger() Logger {
	return NewTextLogger(slog.LevelInfo)
}

func FromSlog(l *slog.Logger) Logger {
	return &slogLogger{l: l}
}

func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }

func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{l: s.l.With(args...)}
}

type ctxKeyLogger struct{}

func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger{}, logger)
}

func LoggerFromContext(ctx context.Context) (Logger, bool) {
	l, ok := ctx.Value(ctxKeyLogger{}).(Logger)
	return l, ok
}
