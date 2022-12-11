package webrtc

import (
	"github.com/pion/logging"
	"github.com/rs/zerolog"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type PionLog struct {
	log *logger.Logger
}

const trace = zerolog.Level(logger.TraceLevel)

func NewPionLogger(root *logger.Logger, level int) *PionLog {
	return &PionLog{log: root.Extend(root.Level(zerolog.Level(level)).With())}
}

func (p PionLog) NewLogger(scope string) logging.LeveledLogger {
	return PionLog{log: p.log.Extend(p.log.With().Str("mod", scope))}
}

func (p PionLog) Debug(msg string)                  { p.log.Debug().Msg(msg) }
func (p PionLog) Debugf(format string, args ...any) { p.log.Debug().Msgf(format, args...) }
func (p PionLog) Error(msg string)                  { p.log.Error().Msg(msg) }
func (p PionLog) Errorf(format string, args ...any) { p.log.Error().Msgf(format, args...) }
func (p PionLog) Info(msg string)                   { p.log.Info().Msg(msg) }
func (p PionLog) Infof(format string, args ...any)  { p.log.Info().Msgf(format, args...) }
func (p PionLog) Trace(msg string)                  { p.log.WithLevel(trace).Msg(msg) }
func (p PionLog) Tracef(format string, args ...any) { p.log.WithLevel(trace).Msgf(format, args...) }
func (p PionLog) Warn(msg string)                   { p.log.Warn().Msg(msg) }
func (p PionLog) Warnf(format string, args ...any)  { p.log.Warn().Msgf(format, args...) }
