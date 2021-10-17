package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"
)

// Level defines log levels.
type Level int8

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled

	// TraceLevel defines trace log level.
	TraceLevel Level = -1
	// Values less than TraceLevel are handled as numbers.
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return zerolog.LevelTraceValue
	case DebugLevel:
		return zerolog.LevelDebugValue
	case InfoLevel:
		return zerolog.LevelInfoValue
	case WarnLevel:
		return zerolog.LevelWarnValue
	case ErrorLevel:
		return zerolog.LevelErrorValue
	case FatalLevel:
		return zerolog.LevelFatalValue
	case PanicLevel:
		return zerolog.LevelPanicValue
	case Disabled:
		return "disabled"
	case NoLevel:
		return ""
	}
	return strconv.Itoa(int(l))
}

var pid = os.Getpid()

type Logger struct {
	logger *zerolog.Logger
}

func New(isDebug bool) *Logger {
	logLevel := zerolog.InfoLevel
	if isDebug {
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	logger := zerolog.New(os.Stderr).With().Timestamp().Fields(map[string]interface{}{"pid": pid}).Logger()
	return &Logger{logger: &logger}
}

func NewConsole(isDebug bool, tag string) *Logger {
	logLevel := zerolog.InfoLevel
	if isDebug {
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	zerolog.TimeFieldFormat = time.RFC3339Nano
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05.0000", NoColor: false}
	output.FormatMessage = func(i interface{}) string {
		if output.NoColor {
			return fmt.Sprintf("%s %v", tag, i)
		}
		return fmt.Sprintf("\x1b[%dm%s\x1b[0m %v", 36, tag, i)
	}
	//multi := zerolog.MultiLevelWriter(output, os.Stdout)
	logger := zerolog.New(output).With().
		Int("pid", pid).
		Str("tag", tag).
		Timestamp().Logger()
	return &Logger{&logger}
}

func Default() *Logger { return &Logger{zerolog.DefaultContextLogger} }

// GetLevel returns the current Level of l.
func (l *Logger) GetLevel() Level { return Level(l.logger.GetLevel()) }

// Output duplicates the global logger and sets w as its output.
func (l *Logger) Output(w io.Writer) zerolog.Logger { return l.logger.Output(w) }

// With creates a child logger with the field added to its context.
func (l *Logger) With() zerolog.Context { return l.logger.With() }

// Level creates a child logger with the minimum accepted level set to level.
func (l *Logger) Level(level zerolog.Level) zerolog.Logger { return l.logger.Level(level) }

// Sample returns a logger with the s sampler.
func (l *Logger) Sample(s zerolog.Sampler) zerolog.Logger { return l.logger.Sample(s) }

// Hook returns a logger with the h Hook.
func (l *Logger) Hook(h zerolog.Hook) zerolog.Logger { return l.logger.Hook(h) }

// Debug starts a new message with debug level.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) Debug() *zerolog.Event { return l.logger.Debug() }

// Info starts a new message with info level.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) Info() *zerolog.Event { return l.logger.Info() }

// Warn starts a new message with warn level.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) Warn() *zerolog.Event { return l.logger.Warn() }

// Error starts a new message with error level.
func (l *Logger) Error() *zerolog.Event { return l.logger.Error() }

// Fatal starts a new message with fatal level. The os.Exit(1) function
// is called by the Msg method.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) Fatal() *zerolog.Event { return l.logger.Fatal() }

// Panic starts a new message with panic level. The message is also sent
// to the panic function.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) Panic() *zerolog.Event { return l.logger.Panic() }

// WithLevel starts a new message with level.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) WithLevel(level zerolog.Level) *zerolog.Event { return l.logger.WithLevel(level) }

// Log starts a new message with no level. Setting zerolog.GlobalLevel to
// zerolog.Disabled will still disable events produced by this method.
// You must call Msg on the returned event in order to send the event.
func (l *Logger) Log() *zerolog.Event { return l.logger.Log() }

// Print sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) { l.logger.Print(v...) }

// Printf sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) { l.logger.Printf(format, v...) }

// Ctx returns the Logger associated with the ctx. If no logger
// is associated, a disabled logger is returned.
func (l *Logger) Ctx(ctx context.Context) *Logger { return &Logger{logger: zerolog.Ctx(ctx)} }

func (l *Logger) Wrap(ctx zerolog.Context) *Logger {
	logger := ctx.Logger()
	return &Logger{&logger}
}
