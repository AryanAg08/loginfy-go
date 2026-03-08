package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents log severity
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var levelColors = map[Level]string{
	DEBUG: "\033[36m", // cyan
	INFO:  "\033[32m", // green
	WARN:  "\033[33m", // yellow
	ERROR: "\033[31m", // red
	FATAL: "\033[35m", // magenta
}

const colorReset = "\033[0m"

// Config holds logger configuration
type Config struct {
	Service    string // service name (e.g. "auth-service", "api-gateway")
	Level      Level  // minimum log level
	TimeFormat string // timestamp format (default: RFC3339Nano)
	LogDir     string // directory for session temp files (default: /tmp/lognify)
	UseColor   bool   // enable ANSI color output to stdout
	JSONOutput bool   // output logs in JSON format
}

// Logger is the main structured logger
type Logger struct {
	mu         sync.RWMutex
	config     Config
	writers    []io.Writer
	sessions   map[string]*Session
	timeFormat string
}

// defaultConfig returns sensible defaults
func defaultConfig() Config {
	return Config{
		Service:    "default",
		Level:      INFO,
		TimeFormat: time.RFC3339,
		LogDir:     filepath.Join(os.TempDir(), "lognify"),
		UseColor:   true,
		JSONOutput: false,
	}
}

// New creates a new Logger with the given config
func New(cfg Config) *Logger {
	if cfg.Service == "" {
		cfg.Service = defaultConfig().Service
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = defaultConfig().TimeFormat
	}
	if cfg.LogDir == "" {
		cfg.LogDir = defaultConfig().LogDir
	}

	l := &Logger{
		config:     cfg,
		writers:    []io.Writer{os.Stdout},
		sessions:   make(map[string]*Session),
		timeFormat: cfg.TimeFormat,
	}
	return l
}

// Default returns a logger with default config
func Default() *Logger {
	return New(defaultConfig())
}

// ForService creates a child logger scoped to a specific service
func (l *Logger) ForService(service string) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	child := &Logger{
		config:     l.config,
		writers:    l.writers,
		sessions:   l.sessions,
		timeFormat: l.timeFormat,
	}
	child.config.Service = service
	return child
}

// SetLevel dynamically changes the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Level = level
}

// AddWriter adds an additional output writer (e.g. file, network)
func (l *Logger) AddWriter(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writers = append(l.writers, w)
}

// caller returns file:line of the log call site
func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "???:0"
	}
	// use only the last 2 path segments for readability
	parts := strings.Split(filepath.ToSlash(file), "/")
	if len(parts) > 2 {
		file = strings.Join(parts[len(parts)-2:], "/")
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// entry formats and writes a single log entry
func (l *Logger) entry(level Level, msg string, fields map[string]interface{}) {
	l.mu.RLock()
	if level < l.config.Level {
		l.mu.RUnlock()
		return
	}
	writers := make([]io.Writer, len(l.writers))
	copy(writers, l.writers)
	cfg := l.config
	l.mu.RUnlock()

	ts := time.Now().Format(l.timeFormat)
	loc := caller(3)

	var line string
	if cfg.JSONOutput {
		line = l.formatJSON(ts, level, cfg.Service, loc, msg, fields)
	} else {
		line = l.formatText(ts, level, cfg.Service, loc, msg, fields, cfg.UseColor)
	}

	for _, w := range writers {
		_, _ = fmt.Fprint(w, line)
	}
}

func (l *Logger) formatText(ts string, level Level, service, loc, msg string, fields map[string]interface{}, color bool) string {
	var b strings.Builder

	if color {
		b.WriteString(fmt.Sprintf("%s[%s]%s ", levelColors[level], levelNames[level], colorReset))
	} else {
		b.WriteString(fmt.Sprintf("[%s] ", levelNames[level]))
	}

	b.WriteString(fmt.Sprintf("%s | service=%s | %s | %s", ts, service, loc, msg))

	if len(fields) > 0 {
		b.WriteString(" |")
		for k, v := range fields {
			b.WriteString(fmt.Sprintf(" %s=%v", k, v))
		}
	}
	b.WriteString("\n")
	return b.String()
}

func (l *Logger) formatJSON(ts string, level Level, service, loc, msg string, fields map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`{"timestamp":"%s","level":"%s","service":"%s","caller":"%s","message":"%s"`,
		ts, levelNames[level], service, loc, escapeJSON(msg)))
	for k, v := range fields {
		b.WriteString(fmt.Sprintf(`,"%s":"%v"`, escapeJSON(k), v))
	}
	b.WriteString("}\n")
	return b.String()
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// --- Public log methods ---

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.entry(DEBUG, msg, mergeFields(fields))
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.entry(INFO, msg, mergeFields(fields))
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.entry(WARN, msg, mergeFields(fields))
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.entry(ERROR, msg, mergeFields(fields))
}

func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	l.entry(FATAL, msg, mergeFields(fields))
	os.Exit(1)
}

// Debugf, Infof, Warnf, Errorf, Fatalf for formatted messages
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.entry(DEBUG, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.entry(INFO, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.entry(WARN, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.entry(ERROR, fmt.Sprintf(format, args...), nil)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.entry(FATAL, fmt.Sprintf(format, args...), nil)
	os.Exit(1)
}

func mergeFields(fields []map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	merged := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			merged[k] = v
		}
	}
	return merged
}

// --- Package-level convenience (default logger) ---

var std = Default()

// SetDefault replaces the package-level default logger
func SetDefault(l *Logger) { std = l }

// Package-level functions forward to the default logger
func DebugMsg(msg string, fields ...map[string]interface{}) { std.Debug(msg, fields...) }
func InfoMsg(msg string, fields ...map[string]interface{})  { std.Info(msg, fields...) }
func WarnMsg(msg string, fields ...map[string]interface{})  { std.Warn(msg, fields...) }
func ErrorMsg(msg string, fields ...map[string]interface{}) { std.Error(msg, fields...) }
func FatalMsg(msg string, fields ...map[string]interface{}) { std.Fatal(msg, fields...) }
func Debugf(format string, args ...interface{})             { std.Debugf(format, args...) }
func Infof(format string, args ...interface{})              { std.Infof(format, args...) }
func Warnf(format string, args ...interface{})              { std.Warnf(format, args...) }
func Errorf(format string, args ...interface{})             { std.Errorf(format, args...) }
func Fatalf(format string, args ...interface{})             { std.Fatalf(format, args...) }
