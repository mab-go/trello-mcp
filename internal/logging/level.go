package logging

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Level represents a logging severity level.
type Level uint8

// Log severity levels.
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel
)

func (l Level) String() string {
	switch l {
	case PanicLevel:
		return "PANIC"
	case FatalLevel:
		return "FATAL"
	case ErrorLevel:
		return "ERROR"
	case WarnLevel:
		return "WARN"
	case InfoLevel:
		return "INFO"
	case DebugLevel:
		return "DEBUG"
	default:
		panic(fmt.Sprintf("invalid log level: %[1]d", l))
	}
}

func (l Level) logrusLevel() logrus.Level {
	switch l {
	case PanicLevel:
		return logrus.PanicLevel
	case FatalLevel:
		return logrus.FatalLevel
	case ErrorLevel:
		return logrus.ErrorLevel
	case WarnLevel:
		return logrus.WarnLevel
	case InfoLevel:
		return logrus.InfoLevel
	case DebugLevel:
		return logrus.DebugLevel
	default:
		panic(fmt.Sprintf("invalid log level: %[1]d", l))
	}
}
