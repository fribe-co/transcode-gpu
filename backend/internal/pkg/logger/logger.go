package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// Init initializes the global logger
func Init(level string, pretty bool) {
	var output io.Writer = os.Stdout

	if pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	log = zerolog.New(output).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Logger()
}

// Get returns the global logger
func Get() *zerolog.Logger {
	return &log
}

// Debug logs debug level message
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info logs info level message
func Info() *zerolog.Event {
	return log.Info()
}

// Warn logs warning level message
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs error level message
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal logs fatal level message and exits
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// WithField returns logger with additional field
func WithField(key string, value interface{}) zerolog.Logger {
	return log.With().Interface(key, value).Logger()
}

// WithFields returns logger with additional fields
func WithFields(fields map[string]interface{}) zerolog.Logger {
	ctx := log.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}





