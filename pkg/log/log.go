package log

import (
	"fmt"
	"io"
	"os"

	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Config struct {
	// Level of logs to output.
	Level string `yaml:"level" validate:"required,oneof=debug info error"`
	// Name of the application for which we are logging.
	Name string `yaml:"name" validate:"required"`
}

// Logger can log messages at levels Info, Debug and Error.
type Logger interface {
	Info(format string, args ...any)
	Debug(format string, args ...any)
	Error(format string, args ...any)
}

type logger struct {
	logger kitlog.Logger
}

func New(cfg Config, writer io.Writer) Logger {
	if writer == nil {
		writer = os.Stdout
	}

	syncWriter := kitlog.NewSyncWriter(writer)
	log := kitlog.NewJSONLogger(syncWriter)

	switch cfg.Level {
	case "debug":
		log = level.NewFilter(log, level.AllowDebug())
	case "info":
		log = level.NewFilter(log, level.AllowInfo())
	case "error":
		log = level.NewFilter(log, level.AllowError())
	default:
		log = level.NewFilter(log, level.AllowInfo())
	}

	prefixed := kitlog.WithPrefix(log, tags(cfg.Name)...)

	return &logger{logger: prefixed}
}

func (l *logger) Info(format string, args ...any) {
	logMsg := fmt.Sprintf(format, args...)
	_ = level.Info(l.logger).Log("msg", logMsg)
}

func (l *logger) Debug(format string, args ...any) {
	logMsg := fmt.Sprintf(format, args...)
	_ = level.Debug(l.logger).Log("msg", logMsg)
}

func (l *logger) Error(format string, args ...any) {
	logMsg := fmt.Sprintf(format, args...)
	_ = level.Error(l.logger).Log("msg", logMsg)
}

func tags(name string) []any {
	t := []any{
		"ts", kitlog.DefaultTimestampUTC,
		"app", name,
		"caller", kitlog.Caller(4),
	}

	return t
}
