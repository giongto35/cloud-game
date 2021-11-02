package logger

import (
	"github.com/pion/logging"
	"github.com/rs/zerolog"
)

type PionLogger struct {
	log *Logger
}

func NewPionLogger(root *Logger, level int) *PionLogger {
	return &PionLogger{log: root.Wrap(root.Level(zerolog.Level(level)).With())}
}

func (p PionLogger) NewLogger(scope string) logging.LeveledLogger {
	return PionLogger{log: p.log.Wrap(p.log.With().Str("mod", scope))}
}

func (p PionLogger) Trace(msg string) { p.log.WithLevel(zerolog.Level(TraceLevel)).Msg(msg) }

func (p PionLogger) Tracef(format string, args ...interface{}) {
	p.log.WithLevel(zerolog.Level(TraceLevel)).Msgf(format, args...)
}

func (p PionLogger) Debug(msg string) { p.log.Debug().Msg(msg) }

func (p PionLogger) Debugf(format string, args ...interface{}) { p.log.Debug().Msgf(format, args...) }

func (p PionLogger) Info(msg string) { p.log.Info().Msg(msg) }

func (p PionLogger) Infof(format string, args ...interface{}) { p.log.Info().Msgf(format, args...) }

func (p PionLogger) Warn(msg string) { p.log.Warn().Msg(msg) }

func (p PionLogger) Warnf(format string, args ...interface{}) { p.log.Warn().Msgf(format, args...) }

func (p PionLogger) Error(msg string) { p.log.Error().Msg(msg) }

func (p PionLogger) Errorf(format string, args ...interface{}) { p.log.Error().Msgf(format, args...) }
