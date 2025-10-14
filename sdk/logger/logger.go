package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/jrazmi/envoker/sdk/environment"
)

// Logger is a wrapper around the standard slog.Logger.
type Logger struct {
	*slog.Logger
}

// options holds all configurable settings for the logger.
type options struct {
	level      slog.Level
	output     io.Writer
	addSource  bool
	format     string // "json" or "text"
	timeFormat string // "RFC3339", "Unix", "UnixMilli", or custom format
}

// Config is the exportable configuration struct
type Options struct {
	Level      string `yaml:"level" toml:"level" json:"level" env:"LOG_LEVEL" default:"INFO"`
	Output     string `yaml:"output" toml:"output" json:"output" env:"LOG_OUTPUT" default:"STDOUT"`
	Format     string `yaml:"format" toml:"format" json:"format" env:"LOG_FORMAT" default:"json"`
	TimeFormat string `yaml:"time_format" toml:"time_format" json:"time_format" env:"LOG_TIME_FORMAT" default:"RFC3339"`
}

// Option takes config option and  returns formatted config
type Option func(*options)

func WithLevel(level string) Option {
	return func(o *options) {
		o.level = parseLevel(level)
	}
}
func NewDefault(opts ...Option) *Logger {
	options := Options{
		Level:      "INFO",
		Output:     "STDERR",
		Format:     "json",
		TimeFormat: time.RFC3339,
	}
	return newLogger(options, opts...)
}

func NewStdLogger(logger *Logger, level slog.Level) *log.Logger {
	return slog.NewLogLogger(logger.Logger.Handler(), level)
}

func NewFromEnv(prefix string, opts ...Option) (*Logger, error) {
	var options Options
	if err := environment.ParseEnvTags(prefix, &options); err != nil {
		return nil, fmt.Errorf("parsing logger config: %w", err)
	}
	return newLogger(options, opts...), nil

}

// new creates a new Logger with default settings and applies any given options.
func newLogger(cfg Options, opts ...Option) *Logger {
	level := parseLevel(cfg.Level)
	output := parseOutput(cfg.Output)

	options := &options{
		level:      level,
		output:     output,
		timeFormat: cfg.TimeFormat,
		format:     cfg.Format,
	}
	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Ensure output is set
	if options.output == nil {
		options.output = os.Stdout
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     options.level,
		AddSource: options.addSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Custom time formatting
			if a.Key == slog.TimeKey && options.timeFormat != "" {
				switch options.timeFormat {
				case "Unix":
					return slog.Int64(slog.TimeKey, a.Value.Time().Unix())
				case "UnixMilli":
					return slog.Int64(slog.TimeKey, a.Value.Time().UnixMilli())
				case "RFC3339Nano":
					return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339Nano))
				case "RFC3339":
					return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339))
				case time.RFC3339, time.RFC3339Nano:
					return slog.String(slog.TimeKey, a.Value.Time().Format(options.timeFormat))
				default:
					// Treat as custom format
					return slog.String(slog.TimeKey, a.Value.Time().Format(options.timeFormat))
				}
			}
			return a
		},
	}

	// Create base handler
	var handler slog.Handler
	switch options.format {
	case "text":
		handler = slog.NewTextHandler(options.output, handlerOpts)
	default:
		handler = slog.NewJSONHandler(options.output, handlerOpts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}

}

// Debugf logs a debug message with formatting
func (l *Logger) DebugContextf(ctx context.Context, format string, args ...any) {
	l.DebugContext(ctx, fmt.Sprintf(format, args...))
}

// Infof logs an info message with formatting
func (l *Logger) InfoContextf(ctx context.Context, format string, args ...any) {
	l.InfoContext(ctx, fmt.Sprintf(format, args...))
}

// Warnf logs a warning message with formatting
func (l *Logger) WarnContextf(ctx context.Context, format string, args ...any) {
	l.WarnContext(ctx, fmt.Sprintf(format, args...))
}

// Errorf logs an error message with formatting
func (l *Logger) ErrorContextf(ctx context.Context, format string, args ...any) {
	l.ErrorContext(ctx, fmt.Sprintf(format, args...))
}
