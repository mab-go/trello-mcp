package logging

// Logger is the structured logging interface used throughout the application.
type Logger interface {
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)
	WithField(key string, value any) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger
	Copy() Logger
}
