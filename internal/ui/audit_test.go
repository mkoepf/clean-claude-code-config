package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	err = logger.Log(ActionDelete, "/path/to/file", 1024)
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	expected := "2025-12-06T16:00:00Z DELETE /path/to/file (1.0 KB)\n"
	assert.Equal(t, expected, string(content))
}

func TestAuditLogger_LogMultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	err = logger.Log(ActionDelete, "/path/one", 1024*1024)
	require.NoError(t, err)

	fixedTime = fixedTime.Add(time.Second)
	logger.now = func() time.Time { return fixedTime }

	err = logger.Log(ActionDelete, "/path/two", 2*1024*1024)
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "/path/one")
	assert.Contains(t, lines[1], "/path/two")
}

func TestAuditLogger_AppendsToExistingLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	// Write initial content
	err := os.WriteFile(logPath, []byte("existing entry\n"), 0644)
	require.NoError(t, err)

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	err = logger.Log(ActionDelete, "/new/entry", 512)
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 2)
	assert.Equal(t, "existing entry", lines[0])
	assert.Contains(t, lines[1], "/new/entry")
}

func TestAuditLogger_CreatesParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "nested", "dir", "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	err = logger.Log(ActionDelete, "/some/path", 100)
	require.NoError(t, err)

	assert.FileExists(t, logPath)
}

func TestAuditLogger_FormatsSizesCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	testCases := []struct {
		size     int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{48 * 1024 * 1024, "48.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}

	for _, tc := range testCases {
		err = logger.Log(ActionDelete, "/test", tc.size)
		require.NoError(t, err)
	}

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	for _, tc := range testCases {
		assert.Contains(t, string(content), tc.expected)
	}
}

func TestDefaultAuditLogPath(t *testing.T) {
	path := DefaultAuditLogPath("/home/user/.claude")
	assert.Equal(t, "/home/user/.claude/cccc-audit.log", path)
}

func TestAuditLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)

	err = logger.Close()
	require.NoError(t, err)

	// Logging after close should fail
	err = logger.Log(ActionDelete, "/path", 100)
	assert.Error(t, err)
}

func TestAuditLogger_LogWithDetails(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	err = logger.LogWithDetails(ActionModify, "/path/to/settings.json", "removed allow: Bash(git:*), Bash(npm:*)")
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	expected := "2025-12-06T16:00:00Z MODIFY /path/to/settings.json: removed allow: Bash(git:*), Bash(npm:*)\n"
	assert.Equal(t, expected, string(content))
}

func TestAuditLogger_LogWithDetails_MultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	require.NoError(t, err)
	defer logger.Close()

	fixedTime := time.Date(2025, 12, 6, 16, 0, 0, 0, time.UTC)
	logger.now = func() time.Time { return fixedTime }

	err = logger.LogWithDetails(ActionModify, "/project1/.claude/settings.local.json", "removed allow: Bash(git:*)")
	require.NoError(t, err)

	err = logger.LogWithDetails(ActionDelete, "/project2/.claude/settings.local.json", "file empty after removing duplicates")
	require.NoError(t, err)

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "MODIFY")
	assert.Contains(t, lines[0], "removed allow: Bash(git:*)")
	assert.Contains(t, lines[1], "DELETE")
	assert.Contains(t, lines[1], "file empty after removing duplicates")
}
