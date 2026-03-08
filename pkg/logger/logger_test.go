package logger_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer

	l := logger.New(logger.Config{
		Service:  "test-service",
		Level:    logger.DEBUG,
		UseColor: false,
	})
	l.AddWriter(&buf)

	l.Info("hello world")

	output := buf.String()
	if !strings.Contains(output, "[INFO]") {
		t.Errorf("expected [INFO] in output, got: %s", output)
	}
	if !strings.Contains(output, "service=test-service") {
		t.Errorf("expected service=test-service in output, got: %s", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	l := logger.New(logger.Config{
		Service:  "filter-test",
		Level:    logger.WARN,
		UseColor: false,
	})
	l.AddWriter(&buf)

	l.Debug("should not appear")
	l.Info("should not appear either")
	l.Warn("this should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("debug/info messages should have been filtered")
	}
	if !strings.Contains(output, "this should appear") {
		t.Error("warn message should be present")
	}
}

func TestLoggerFields(t *testing.T) {
	var buf bytes.Buffer

	l := logger.New(logger.Config{
		Service:  "field-test",
		Level:    logger.DEBUG,
		UseColor: false,
	})
	l.AddWriter(&buf)

	l.Info("user login", map[string]interface{}{
		"user_id": "u123",
		"ip":      "192.168.1.1",
	})

	output := buf.String()
	if !strings.Contains(output, "user_id=u123") {
		t.Errorf("expected user_id field, got: %s", output)
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer

	l := logger.New(logger.Config{
		Service:    "json-test",
		Level:      logger.DEBUG,
		JSONOutput: true,
	})
	l.AddWriter(&buf)

	l.Info("json log")

	output := buf.String()
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Errorf("expected JSON level, got: %s", output)
	}
	if !strings.Contains(output, `"service":"json-test"`) {
		t.Errorf("expected JSON service, got: %s", output)
	}
}

func TestForService(t *testing.T) {
	var buf bytes.Buffer

	parent := logger.New(logger.Config{
		Service:  "parent",
		Level:    logger.DEBUG,
		UseColor: false,
	})
	parent.AddWriter(&buf)

	child := parent.ForService("child-service")
	child.Info("from child")

	output := buf.String()
	if !strings.Contains(output, "service=child-service") {
		t.Errorf("expected child service name, got: %s", output)
	}
}

func TestSessionCreation(t *testing.T) {
	tmpDir := t.TempDir()

	l := logger.New(logger.Config{
		Service:  "session-test",
		Level:    logger.DEBUG,
		LogDir:   tmpDir,
		UseColor: false,
	})

	sess, err := l.StartSession("sess-001")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	if sess.ID != "sess-001" {
		t.Errorf("expected session ID sess-001, got %s", sess.ID)
	}

	// verify file was created
	if _, err := os.Stat(sess.FilePath); os.IsNotExist(err) {
		t.Error("session log file was not created")
	}

	sess.Info("session log entry", map[string]interface{}{"key": "value"})

	if err := sess.End(); err != nil {
		t.Fatalf("failed to end session: %v", err)
	}

	// verify file contents
	data, err := os.ReadFile(sess.FilePath)
	if err != nil {
		t.Fatalf("failed to read session file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "SESSION STARTED") {
		t.Error("missing session header")
	}
	if !strings.Contains(content, "session log entry") {
		t.Error("missing session log entry")
	}
	if !strings.Contains(content, "SESSION ENDED") {
		t.Error("missing session footer")
	}
}

func TestDuplicateSession(t *testing.T) {
	tmpDir := t.TempDir()

	l := logger.New(logger.Config{
		Service: "dup-test",
		LogDir:  tmpDir,
	})

	_, err := l.StartSession("dup-001")
	if err != nil {
		t.Fatalf("first session should succeed: %v", err)
	}

	_, err = l.StartSession("dup-001")
	if err == nil {
		t.Error("duplicate session should return error")
	}
}

func TestCloseAllSessions(t *testing.T) {
	tmpDir := t.TempDir()

	l := logger.New(logger.Config{
		Service: "close-all-test",
		LogDir:  tmpDir,
	})

	_, _ = l.StartSession("s1")
	_, _ = l.StartSession("s2")
	_, _ = l.StartSession("s3")

	if l.ActiveSessions() != 3 {
		t.Errorf("expected 3 active sessions, got %d", l.ActiveSessions())
	}

	l.CloseAllSessions()

	if l.ActiveSessions() != 0 {
		t.Errorf("expected 0 active sessions after close, got %d", l.ActiveSessions())
	}
}

func TestSessionFileNaming(t *testing.T) {
	tmpDir := t.TempDir()

	l := logger.New(logger.Config{
		Service: "naming-test",
		LogDir:  tmpDir,
	})

	sess, _ := l.StartSession("abc-123")
	defer sess.End()

	filename := filepath.Base(sess.FilePath)
	if !strings.HasPrefix(filename, "naming-test_abc-123_") {
		t.Errorf("unexpected filename format: %s", filename)
	}
	if !strings.HasSuffix(filename, ".log") {
		t.Errorf("expected .log extension: %s", filename)
	}
}

func TestServiceLogger(t *testing.T) {
	var buf bytes.Buffer

	sl := logger.NewServiceLogger("auth-service", logger.Config{
		Level:    logger.DEBUG,
		UseColor: false,
	})
	sl.Logger().AddWriter(&buf)

	sl.Info("user authenticated", map[string]interface{}{
		"user": "john",
	})

	output := buf.String()
	if !strings.Contains(output, "service=auth-service") {
		t.Errorf("expected service name, got: %s", output)
	}
}
