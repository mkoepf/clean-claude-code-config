package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AuditLogger handles audit trail logging for cleanup operations.
type AuditLogger struct {
	file   *os.File
	now    func() time.Time
	closed bool
}

// NewAuditLogger creates a new audit logger that writes to the specified path.
// Creates parent directories if they don't exist.
func NewAuditLogger(path string) (*AuditLogger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{
		file: file,
		now:  time.Now,
	}, nil
}

// Log writes an audit entry for a cleanup action.
// Format: 2025-12-06T16:00:00Z DELETE /path/to/file (48 MB)
func (l *AuditLogger) Log(action Action, path string, size int64) error {
	if l.closed {
		return fmt.Errorf("audit logger is closed")
	}

	timestamp := l.now().UTC().Format(time.RFC3339)
	sizeStr := FormatSize(size)

	entry := fmt.Sprintf("%s %s %s (%s)\n", timestamp, action, path, sizeStr)

	_, err := l.file.WriteString(entry)
	return err
}

// Close closes the audit log file.
func (l *AuditLogger) Close() error {
	l.closed = true
	return l.file.Close()
}

// DefaultAuditLogPath returns the default audit log path for a given Claude home directory.
func DefaultAuditLogPath(claudeHome string) string {
	return filepath.Join(claudeHome, "ccc-audit.log")
}
