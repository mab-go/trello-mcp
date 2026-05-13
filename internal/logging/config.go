// Package logging provides a structured logging interface backed by logrus.
package logging

import (
	"io"
)

// Config holds logger settings.
type Config struct {
	Level  Level
	Output io.Writer
}
