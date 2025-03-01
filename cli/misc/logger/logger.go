package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
)

type plainTextHandler struct {
	writer io.Writer
	level  slog.Level
}

func (h *plainTextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *plainTextHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *plainTextHandler) WithGroup(string) slog.Handler      { return h }

func (h *plainTextHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}
	msg := strings.TrimPrefix(r.Message, "msg=")
	fmt.Fprintln(h.writer, msg)
	return nil
}

var defaultHandler *plainTextHandler

func Initialize() {
	defaultHandler = &plainTextHandler{
		writer: os.Stdout,
		level:  slog.LevelInfo,
	}

	if constants.Version == "development" {
		defaultHandler.level = slog.LevelDebug
	}

	logger := slog.New(defaultHandler)
	slog.SetDefault(logger)
}

func Debug(msg string) {
	slog.Debug(fmt.Sprintf("DEBUG: %s", msg))
}

func Info(msg string) {
	slog.Info(msg)
}

func Infof(msg string, args ...interface{}) {
	slog.Info(fmt.Sprintf(msg, args...))
}

func Warning(msg string) {
	slog.Warn(msg)
}

func Error(msg string, err error) {
	if err != nil {
		slog.Debug(err.Error())
	}
	slog.Error(msg)
}
