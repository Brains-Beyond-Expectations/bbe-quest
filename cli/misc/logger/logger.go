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

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
)

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

type plainTextHandler struct {
	writer  io.Writer
	level   slog.Level
	colored bool
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

	if !h.colored {
		switch r.Level {
		case slog.LevelDebug:
			fmt.Fprintf(h.writer, "DEBUG  %s\n", msg)
		case slog.LevelWarn:
			fmt.Fprintf(h.writer, "WARN   %s\n", msg)
		case slog.LevelError:
			fmt.Fprintf(h.writer, "ERROR  %s\n", msg)
		default:
			fmt.Fprintln(h.writer, msg)
		}
		return nil
	}

	switch r.Level {
	case slog.LevelDebug:
		fmt.Fprintf(h.writer, "%sDEBUG%s  %s\n", colorCyan, colorReset, msg)
	case slog.LevelWarn:
		fmt.Fprintf(h.writer, "%sWARN%s   %s\n", colorYellow, colorReset, msg)
	case slog.LevelError:
		fmt.Fprintf(h.writer, "%sERROR%s  %s\n", colorRed, colorReset, msg)
	default:
		fmt.Fprintf(h.writer, "%s%s%s\n", colorGreen, msg, colorReset)
	}
	return nil
}

var defaultHandler *plainTextHandler

func Initialize() {
	out := os.Stdout
	defaultHandler = &plainTextHandler{
		writer:  out,
		level:   slog.LevelInfo,
		colored: isTerminal(out),
	}

	if constants.Version == "development" {
		defaultHandler.level = slog.LevelDebug
	}

	logger := slog.New(defaultHandler)
	slog.SetDefault(logger)
}

func Debug(msg string) {
	slog.Debug(msg)
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
	if msg != "" {
		slog.Error(msg)
	}
}
