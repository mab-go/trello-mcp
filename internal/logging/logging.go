package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Fields is a map of key-value pairs for structured log entries.
type Fields map[string]any

type logrusLogger struct {
	entry *logrus.Entry
}

// NewLogger creates a Logger with the given configuration.
func NewLogger(config Config) Logger {
	log := logrus.New()
	log.Level = config.Level.logrusLevel()

	output := config.Output
	if output == nil {
		output = os.Stderr
	}

	log.SetOutput(output)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	return &logrusLogger{entry: logrus.NewEntry(log)}
}

// NewDefaultLogger creates a Logger with the default configuration.
func NewDefaultLogger() Logger {
	return NewLogger(defaultConfig)
}

var _ Logger = (*logrusLogger)(nil)

func (log *logrusLogger) Debug(args ...any) {
	log.entry.Debug(args...)
}

func (log *logrusLogger) Info(args ...any) {
	log.entry.Info(args...)
}

func (log *logrusLogger) Warn(args ...any) {
	log.entry.Warn(args...)
}

func (log *logrusLogger) Error(args ...any) {
	log.entry.Error(args...)
}

func (log *logrusLogger) Fatal(args ...any) {
	log.entry.Fatal(args...)
}

func (log *logrusLogger) Panic(args ...any) {
	log.entry.Panic(args...)
}

func (log *logrusLogger) WithField(key string, value any) Logger {
	return &logrusLogger{entry: log.entry.WithField(key, value)}
}

func (log *logrusLogger) WithFields(fields Fields) Logger {
	return &logrusLogger{entry: log.entry.WithFields(logrus.Fields(fields))}
}

func (log *logrusLogger) WithError(err error) Logger {
	return &logrusLogger{entry: log.entry.WithError(err)}
}

func (log *logrusLogger) Copy() Logger {
	return &logrusLogger{entry: log.entry.Dup()}
}
