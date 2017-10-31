package zap

import (
	"github.com/iqoption/rpcng/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// mute logger struct
type l struct {
	zap *zap.Logger
}

// Mute logger constructor
func New(z *zap.Logger) logger.Logger {
	return &l{
		zap: z,
	}
}

// Debug message
func (l *l) Debug(message string, vars ...interface{}) {
	var ce *zapcore.CheckedEntry
	if ce = l.zap.Check(zap.DebugLevel, message); ce == nil {
		return
	}

	ce.Write(l.format(vars)...)
}

// Info message
func (l *l) Info(message string, vars ...interface{}) {
	var ce *zapcore.CheckedEntry
	if ce = l.zap.Check(zap.InfoLevel, message); ce == nil {
		return
	}

	ce.Write(l.format(vars)...)
}

// Error message
func (l *l) Error(message string, vars ...interface{}) {
	var ce *zapcore.CheckedEntry
	if ce = l.zap.Check(zap.ErrorLevel, message); ce == nil {
		return
	}

	ce.Write(l.format(vars)...)
}

// Format entries
func (l *l) format(vars []interface{}) []zapcore.Field {
	var (
		ok     bool
		key    string
		fields = make([]zapcore.Field, 0, len(vars)/2)
	)

	for i := 0; i < len(vars); i += 2 {
		if key, ok = vars[i].(string); !ok {
			return nil
		}

		if i+1 >= len(vars) {
			return nil
		}

		fields = append(fields, zap.Any(key, vars[i+1]))
	}

	return fields
}
