package logger

import (
	"fmt"
	"net/http"
	"time"
)

// ServiceLogger wraps a Logger with a fixed service name and optional session.
// This is the recommended way to use the logger within a specific microservice.
type ServiceLogger struct {
	logger  *Logger
	service string
}

// NewServiceLogger creates a logger bound to a specific service name.
// All logs produced will include the service identifier.
func NewServiceLogger(service string, cfg ...Config) *ServiceLogger {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	} else {
		c = defaultConfig()
	}
	c.Service = service

	return &ServiceLogger{
		logger:  New(c),
		service: service,
	}
}

// Logger returns the underlying Logger
func (sl *ServiceLogger) Logger() *Logger { return sl.logger }

// Debug, Info, Warn, Error, Fatal with service context
func (sl *ServiceLogger) Debug(msg string, fields ...map[string]interface{}) {
	sl.logger.Debug(msg, fields...)
}

func (sl *ServiceLogger) Info(msg string, fields ...map[string]interface{}) {
	sl.logger.Info(msg, fields...)
}

func (sl *ServiceLogger) Warn(msg string, fields ...map[string]interface{}) {
	sl.logger.Warn(msg, fields...)
}

func (sl *ServiceLogger) Error(msg string, fields ...map[string]interface{}) {
	sl.logger.Error(msg, fields...)
}

func (sl *ServiceLogger) Fatal(msg string, fields ...map[string]interface{}) {
	sl.logger.Fatal(msg, fields...)
}

// Debugf, Infof, Warnf, Errorf for formatted messages
func (sl *ServiceLogger) Debugf(format string, args ...interface{}) {
	sl.logger.Debugf(format, args...)
}
func (sl *ServiceLogger) Infof(format string, args ...interface{}) { sl.logger.Infof(format, args...) }
func (sl *ServiceLogger) Warnf(format string, args ...interface{}) { sl.logger.Warnf(format, args...) }
func (sl *ServiceLogger) Errorf(format string, args ...interface{}) {
	sl.logger.Errorf(format, args...)
}

// StartSession creates a session scoped to this service
func (sl *ServiceLogger) StartSession(sessionID string) (*Session, error) {
	return sl.logger.StartSession(sessionID)
}

// HTTPMiddleware returns an http.Handler middleware that logs request details
func (sl *ServiceLogger) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		sl.Info("request started", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		})

		next.ServeHTTP(wrapped, r)

		sl.Info("request completed", map[string]interface{}{
			"method":   r.Method,
			"path":     r.URL.Path,
			"status":   wrapped.statusCode,
			"duration": fmt.Sprintf("%dms", time.Since(start).Milliseconds()),
		})
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
