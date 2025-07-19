package log

import (
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SentryConfig contains the configuration for the sentry sink
type SentryConfig struct {
	SentryDsn                string         `yaml:"sentry_dsn"`
	Debug                    bool           `yaml:"debug"`
	Version                  string         `yaml:"version"`
	Environment              string         `yaml:"environment"`
	SentryLogLevelsToCapture []sentry.Level `yaml:"sentry_log_levels_to_capture"`
}

// Logger wraps logr.Logger to provide additional methods
type Logger struct {
	logr.Logger
	underlyingZap *zap.Logger
	cleanupFuncs  []func() error
}

var globalLevel = zap.NewAtomicLevelAt(zap.InfoLevel)

// SetLevel sets the global logging level
func SetLevel(level int) {
	globalLevel.SetLevel(zapcore.Level(-1 * level))
}

// New creates a new logger with the given name (with no default outputs)
func New(name string) (*Logger, func() error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create a no-op core that doesn't output logs anywhere
	noopCore := zapcore.NewNopCore()

	// Build options for the logger - use a more appropriate CallerSkip value
	zapOpts := []zap.Option{
		zap.AddCaller(),
		zap.Fields(zap.String("component", name)),
	}

	// Create the logger with a noop core (no outputs)
	zapLogger := zap.New(noopCore, zapOpts...)

	// Create the logger instance
	logger := &Logger{
		Logger:        zapr.NewLogger(zapLogger),
		underlyingZap: zapLogger,
		cleanupFuncs:  []func() error{zapLogger.Sync},
	}

	// Return the cleanup function
	cleanup := func() error {
		var firstErr error
		// Run all cleanup functions in reverse order (LIFO)
		for i := len(logger.cleanupFuncs) - 1; i >= 0; i-- {
			if err := logger.cleanupFuncs[i](); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	return logger, cleanup
}

// SinkOption is a functional option for configuring a logger sink
type SinkOption func(*sinkOptions)

type sinkOptions struct {
	output io.Writer
	level  zap.AtomicLevel
}

// WithConsoleSink returns a sink option for console output
func WithConsoleSink(w io.Writer) SinkOption {
	return func(o *sinkOptions) {
		o.output = w
	}
}

// AddConsoleSink adds a console sink to the logger
func (l *Logger) AddConsoleSink(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	zl := l.underlyingZap

	// Create a new console core
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(w), globalLevel)

	// Create a new logger with the original core and the console core
	newLogger := zl.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, consoleCore)
	}))

	// Update the current logger
	l.Logger = zapr.NewLogger(newLogger)
	l.underlyingZap = newLogger

	// Add the new cleanup function
	l.cleanupFuncs = append(l.cleanupFuncs, newLogger.Sync)

	return nil
}

// AddSink adds a sink to the logger
func (l *Logger) AddSink(option SinkOption) error {
	if option == nil {
		return fmt.Errorf("sink option cannot be nil")
	}

	zapLogger, ok := l.GetSink().(zapr.Underlier)
	if !ok {
		return fmt.Errorf("failed to get underlying zap logger")
	}

	zl := zapLogger.GetUnderlying()

	opts := &sinkOptions{
		output: os.Stdout,
		level:  globalLevel,
	}
	option(opts)

	if opts.output == nil {
		return fmt.Errorf("output writer cannot be nil")
	}

	// Create a new console core
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(opts.output), opts.level)

	// Create a new logger with the original core and the console core
	newLogger := zl.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, consoleCore)
	}))

	// Update the current logger
	l.Logger = zapr.NewLogger(newLogger)
	l.underlyingZap = newLogger

	// Add the new cleanup function
	l.cleanupFuncs = append(l.cleanupFuncs, newLogger.Sync)

	return nil
}

// AddSentrySink adds a Sentry sink to the logger using configuration from the provided config
func (l *Logger) AddSentrySink(config SentryConfig) error {
	zl := l.underlyingZap

	beforeSend := func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		if slices.Contains(config.SentryLogLevelsToCapture, event.Level) {
			return event
		}
		return nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:                   config.SentryDsn,
		Debug:                 config.Debug,
		AttachStacktrace:      true,
		EnableTracing:         true,
		TracesSampleRate:      1.0,
		Release:               config.Version,
		Environment:           config.Environment,
		BeforeSend:            beforeSend,
		BeforeSendTransaction: beforeSend,
	})
	if err != nil {
		return err
	}

	cfg := zapsentry.Configuration{
		Level:             zapcore.DebugLevel,
		EnableBreadcrumbs: true,
		BreadcrumbLevel:   zapcore.DebugLevel,
		Tags:              map[string]string{"component": "system"},
	}

	sentryCore, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()))
	if err != nil {
		return err
	}

	newLogger := zl.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentryCore)
	}))

	// Create a sentry flush cleanup function
	sentryFlush := func() error {
		sentry.Flush(2 * time.Second)
		return nil
	}

	// Update the current logger
	l.Logger = zapr.NewLogger(newLogger)
	l.underlyingZap = newLogger

	// Add the new cleanup functions
	l.cleanupFuncs = append(l.cleanupFuncs, newLogger.Sync, sentryFlush)

	return nil
}

// AsLogr returns the underlying logr.Logger for compatibility with libraries that expect logr.Logger
func (l *Logger) AsLogr() logr.Logger {
	return l.Logger
}
