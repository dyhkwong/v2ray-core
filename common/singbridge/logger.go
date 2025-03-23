package singbridge

import (
	"context"

	"github.com/sagernet/sing/common/logger"

	"github.com/v2fly/v2ray-core/v5/common/errors"
	"github.com/v2fly/v2ray-core/v5/common/session"
)

var _ logger.ContextLogger = (*loggerWrapper)(nil)

type loggerWrapper struct {
	newError func(values ...any) *errors.Error
}

func NewLoggerWrapper(newErrorFunc func(values ...any) *errors.Error) *loggerWrapper {
	return &loggerWrapper{
		newErrorFunc,
	}
}

func (l *loggerWrapper) Trace(args ...any) {
}

func (l *loggerWrapper) Debug(args ...any) {
	l.newError(args...).AtDebug().WriteToLog()
}

func (l *loggerWrapper) Info(args ...any) {
	l.newError(args...).AtInfo().WriteToLog()
}

func (l *loggerWrapper) Warn(args ...any) {
	l.newError(args...).AtWarning().WriteToLog()
}

func (l *loggerWrapper) Error(args ...any) {
	l.newError(args...).AtError().WriteToLog()
}

func (l *loggerWrapper) Fatal(args ...any) {
	l.newError(args...).AtError().WriteToLog()
}

func (l *loggerWrapper) Panic(args ...any) {
	l.newError(args...).AtError().WriteToLog()
}

func (l *loggerWrapper) TraceContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) DebugContext(ctx context.Context, args ...any) {
	l.newError(args...).AtDebug().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) InfoContext(ctx context.Context, args ...any) {
	l.newError(args...).AtInfo().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) WarnContext(ctx context.Context, args ...any) {
	l.newError(args...).AtWarning().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) ErrorContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) FatalContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}

func (l *loggerWrapper) PanicContext(ctx context.Context, args ...any) {
	l.newError(args...).AtError().WriteToLog(session.ExportIDToError(ctx))
}
