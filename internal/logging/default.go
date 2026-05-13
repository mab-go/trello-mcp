package logging

var defaultConfig Config
var defaultLogger Logger

func init() {
	defaultConfig = Config{
		Level: DebugLevel,
	}

	defaultLogger = NewDefaultLogger()
}

// SetDefaultConfig updates the default logger configuration.
func SetDefaultConfig(config Config) {
	defaultConfig = config
	defaultLogger = NewDefaultLogger()
}

// Debug logs at debug level using the default logger.
func Debug(args ...any) {
	defaultLogger.Debug(args...)
}

// Info logs at info level using the default logger.
func Info(args ...any) {
	defaultLogger.Info(args...)
}

// Warn logs at warn level using the default logger.
func Warn(args ...any) {
	defaultLogger.Warn(args...)
}

// Error logs at error level using the default logger.
func Error(args ...any) {
	defaultLogger.Error(args...)
}

// Fatal logs at fatal level using the default logger, then exits.
func Fatal(args ...any) {
	defaultLogger.Fatal(args...)
}

// Panic logs at panic level using the default logger, then panics.
func Panic(args ...any) {
	defaultLogger.Panic(args...)
}

// WithField returns a Logger with a single key-value field added.
func WithField(key string, value any) Logger {
	return defaultLogger.WithField(key, value)
}

// WithFields returns a Logger with multiple key-value fields added.
func WithFields(fields Fields) Logger {
	return defaultLogger.WithFields(fields)
}

// WithError returns a Logger with an error field added.
func WithError(err error) Logger {
	return defaultLogger.WithError(err)
}
