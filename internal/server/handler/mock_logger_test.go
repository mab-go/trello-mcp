package handler

import (
	"sync"

	"github.com/mab-go/logging"
)

type logEntry struct {
	Level  string
	Event  logging.Event
	Fields logging.Fields
}

func (e *logEntry) HasField(key string) bool {
	_, ok := e.Fields[key]
	return ok
}

type recordingLogger struct {
	mu      *sync.Mutex
	entries *[]logEntry
	fields  logging.Fields
}

func newRecordingLogger() *recordingLogger {
	return &recordingLogger{
		mu:      &sync.Mutex{},
		entries: &[]logEntry{},
		fields:  logging.Fields{},
	}
}

func (r *recordingLogger) record(level string, args ...any) {
	entry := logEntry{
		Level:  level,
		Fields: copyFields(r.fields),
	}
	if len(args) > 0 {
		if ev, ok := args[0].(logging.Event); ok {
			entry.Event = ev
		}
	}
	r.mu.Lock()
	*r.entries = append(*r.entries, entry)
	r.mu.Unlock()
}

func (r *recordingLogger) Debug(args ...any) { r.record("DEBUG", args...) }
func (r *recordingLogger) Info(args ...any)  { r.record("INFO", args...) }
func (r *recordingLogger) Warn(args ...any)  { r.record("WARN", args...) }
func (r *recordingLogger) Error(args ...any) { r.record("ERROR", args...) }
func (r *recordingLogger) Fatal(args ...any) { r.record("FATAL", args...) }
func (r *recordingLogger) Panic(args ...any) { r.record("PANIC", args...) }

func (r *recordingLogger) WithField(key string, value any) logging.Logger {
	f := copyFields(r.fields)
	f[key] = value
	return &recordingLogger{mu: r.mu, entries: r.entries, fields: f}
}

func (r *recordingLogger) WithFields(fields logging.Fields) logging.Logger {
	f := copyFields(r.fields)
	for k, v := range fields {
		f[k] = v
	}
	return &recordingLogger{mu: r.mu, entries: r.entries, fields: f}
}

func (r *recordingLogger) WithError(err error) logging.Logger {
	if err == nil {
		return r
	}
	return r.WithField("error", err.Error())
}

func (r *recordingLogger) Copy() logging.Logger {
	return &recordingLogger{mu: r.mu, entries: r.entries, fields: copyFields(r.fields)}
}

func (r *recordingLogger) HasEntry(level string, event logging.Event) bool {
	return r.Entry(level, event) != nil
}

func (r *recordingLogger) Entry(level string, event logging.Event) *logEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range *r.entries {
		if (*r.entries)[i].Level == level && (*r.entries)[i].Event == event {
			return &(*r.entries)[i]
		}
	}
	return nil
}

func (r *recordingLogger) Entries() []logEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]logEntry, len(*r.entries))
	copy(out, *r.entries)
	return out
}

func copyFields(src logging.Fields) logging.Fields {
	dst := make(logging.Fields, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
