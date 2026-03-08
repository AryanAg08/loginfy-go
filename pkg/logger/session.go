package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Session represents a logging session with its own temporary log file.
// Each session captures logs from a specific workflow/request lifecycle.
type Session struct {
	ID        string
	Service   string
	StartedAt time.Time
	FilePath  string

	mu     sync.Mutex
	file   *os.File
	logger *Logger
	closed bool
}

// StartSession creates a new session with a dedicated temp log file.
// The file is created under the logger's LogDir as: <service>_<sessionID>_<timestamp>.log
func (l *Logger) StartSession(sessionID string) (*Session, error) {
	l.mu.Lock()
	// NOTE: mu.Unlock is called manually before the Info log call at the end

	if _, exists := l.sessions[sessionID]; exists {
		l.mu.Unlock()
		return nil, fmt.Errorf("session %s already exists", sessionID)
	}

	// ensure log directory exists
	if err := os.MkdirAll(l.config.LogDir, 0755); err != nil {
		l.mu.Unlock()
		return nil, fmt.Errorf("failed to create log dir %s: %w", l.config.LogDir, err)
	}

	now := time.Now()
	filename := fmt.Sprintf("%s_%s_%s.log",
		l.config.Service,
		sessionID,
		now.Format("20060102_150405"),
	)
	filePath := filepath.Join(l.config.LogDir, filename)

	f, err := os.Create(filePath)
	if err != nil {
		l.mu.Unlock()
		return nil, fmt.Errorf("failed to create session log file: %w", err)
	}

	sess := &Session{
		ID:        sessionID,
		Service:   l.config.Service,
		StartedAt: now,
		FilePath:  filePath,
		file:      f,
		logger:    l,
	}

	l.sessions[sessionID] = sess

	// write session header
	header := fmt.Sprintf("=== SESSION STARTED ===\nSession ID : %s\nService    : %s\nStarted At : %s\nLog File   : %s\n========================\n\n",
		sessionID, l.config.Service, now.Format(l.timeFormat), filePath)
	_, _ = f.WriteString(header)

	// release lock before logging to avoid deadlock (entry acquires RLock)
	l.mu.Unlock()
	l.Info("session started", map[string]interface{}{
		"session_id": sessionID,
		"log_file":   filePath,
	})

	return sess, nil
}

// Log writes a log entry to the session's temp file with timestamp and service info
func (s *Session) Log(level Level, msg string, fields ...map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.file == nil {
		return
	}

	ts := time.Now().Format(s.logger.timeFormat)
	loc := caller(2)
	merged := mergeFields(fields)

	line := fmt.Sprintf("[%s] %s | service=%s | session=%s | %s | %s",
		levelNames[level], ts, s.Service, s.ID, loc, msg)

	if len(merged) > 0 {
		line += " |"
		for k, v := range merged {
			line += fmt.Sprintf(" %s=%v", k, v)
		}
	}
	line += "\n"

	_, _ = s.file.WriteString(line)

	// also send to the parent logger's writers
	s.logger.entry(level, msg, mergeSessionFields(s.ID, merged))
}

func mergeSessionFields(sessionID string, fields map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{"session_id": sessionID}
	for k, v := range fields {
		result[k] = v
	}
	return result
}

// Debug, Info, Warn, Error convenience methods on session
func (s *Session) Debug(msg string, fields ...map[string]interface{}) { s.Log(DEBUG, msg, fields...) }
func (s *Session) Info(msg string, fields ...map[string]interface{})  { s.Log(INFO, msg, fields...) }
func (s *Session) Warn(msg string, fields ...map[string]interface{})  { s.Log(WARN, msg, fields...) }
func (s *Session) Error(msg string, fields ...map[string]interface{}) { s.Log(ERROR, msg, fields...) }

// End closes the session, flushes and closes the temp file
func (s *Session) End() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	duration := time.Since(s.StartedAt)
	footer := fmt.Sprintf("\n========================\nSession ID : %s\nDuration   : %s\nEnded At   : %s\n=== SESSION ENDED ===\n",
		s.ID, duration, time.Now().Format(s.logger.timeFormat))
	_, _ = s.file.WriteString(footer)

	s.closed = true
	err := s.file.Close()

	// remove from parent logger's session map
	s.logger.mu.Lock()
	delete(s.logger.sessions, s.ID)
	s.logger.mu.Unlock()

	s.logger.Info("session ended", map[string]interface{}{
		"session_id": s.ID,
		"duration":   duration.String(),
		"log_file":   s.FilePath,
	})

	return err
}

// GetSession returns an active session by ID
func (l *Logger) GetSession(sessionID string) (*Session, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	s, ok := l.sessions[sessionID]
	return s, ok
}

// ActiveSessions returns the count of active sessions
func (l *Logger) ActiveSessions() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.sessions)
}

// CloseAllSessions ends all active sessions (useful for graceful shutdown)
func (l *Logger) CloseAllSessions() {
	l.mu.RLock()
	ids := make([]string, 0, len(l.sessions))
	for id := range l.sessions {
		ids = append(ids, id)
	}
	l.mu.RUnlock()

	for _, id := range ids {
		if sess, ok := l.GetSession(id); ok {
			_ = sess.End()
		}
	}
}

// --- Package-level session helpers ---

func StartSession(sessionID string) (*Session, error) { return std.StartSession(sessionID) }
func GetSession(sessionID string) (*Session, bool)     { return std.GetSession(sessionID) }
func ActiveSessions() int                              { return std.ActiveSessions() }
func CloseAllSessions()                                { std.CloseAllSessions() }
