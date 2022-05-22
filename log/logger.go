package log

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the debug severity or above.
	logrus.SetLevel(logrus.DebugLevel)

	// Report callers from all packages
	logrus.SetReportCaller(true)
}

// Logger implements log.Logger
type Logger struct {
	*logrus.Entry
}

// Log global logger
var Log = Wrap(logrus.WithFields(logrus.Fields{}))

// Wrap wrap *logrus.Entry to log.Logger
func Wrap(logrusEntry *logrus.Entry) *Logger {
	return &Logger{logrusEntry}
}

// LoggerWithSpan spwan new logger with trace span
func LoggerWithSpan(ctx context.Context) *Logger {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return Log
	}

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	newLogrus := Log.WithFields(logrus.Fields{
		"trace": traceID,
		"span":  spanID,
	})

	return Wrap(newLogrus)
}

// WithSpan add trace span to logger
func WithSpan(ctx context.Context) *Logger {
	return LoggerWithSpan(ctx)
}

// WithError add error to logger
func WithError(err error) *Logger {
	return Wrap(Log.WithError(err))
}

// WithDefaultFields add default fields to logger
func WithDefaultFields(key string, value interface{}) {
	Log = Wrap(Log.WithField(key, value))
}

// SetLevel set log level
func SetLevel(level string) {
	switch strings.ToLower(level) {
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		Log.Warnf("unknown log level %s, set loglevel Debug", level)
		logrus.SetLevel(logrus.DebugLevel)
	}
}

// ---------------------------------------
// from logrus

func WithField(key string, value interface{}) *Logger {
	return Wrap(Log.WithField(key, value))
}

func Trace(args ...interface{}) {
	Log.Log(logrus.TraceLevel, args...)
}

func Debug(args ...interface{}) {
	Log.Log(logrus.DebugLevel, args...)
}

func Print(args ...interface{}) {
	Log.Info(args...)
}

func Info(args ...interface{}) {
	Log.Log(logrus.InfoLevel, args...)
}

func Warn(args ...interface{}) {
	Log.Log(logrus.WarnLevel, args...)
}

func Warning(args ...interface{}) {
	Log.Warn(args...)
}

func Error(args ...interface{}) {
	Log.Log(logrus.ErrorLevel, args...)
}

func Fatal(args ...interface{}) {
	Log.Log(logrus.FatalLevel, args...)
	Log.Logger.Exit(1)
}

func Panic(args ...interface{}) {
	Log.Log(logrus.PanicLevel, args...)
	panic(fmt.Sprint(args...))
}

func Logf(level logrus.Level, format string, args ...interface{}) {
	if Log.Logger.IsLevelEnabled(level) {
		Log.Log(level, fmt.Sprintf(format, args...))
	}
}

func Tracef(format string, args ...interface{}) {
	Log.Logf(logrus.TraceLevel, format, args...)
}

func Debugf(format string, args ...interface{}) {
	Log.Logf(logrus.DebugLevel, format, args...)
}

func Infof(format string, args ...interface{}) {
	Log.Logf(logrus.InfoLevel, format, args...)
}

func Printf(format string, args ...interface{}) {
	Log.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Log.Logf(logrus.WarnLevel, format, args...)
}

func Warningf(format string, args ...interface{}) {
	Log.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Log.Logf(logrus.ErrorLevel, format, args...)
}

func Fatalf(format string, args ...interface{}) {
	Log.Logf(logrus.FatalLevel, format, args...)
	Log.Logger.Exit(1)
}

func Panicf(format string, args ...interface{}) {
	Log.Logf(logrus.PanicLevel, format, args...)
}
