package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkoepf/cccc/internal/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindOrphans_EmptySessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &claude.Paths{
		Root:        tmpDir,
		Projects:    filepath.Join(tmpDir, "projects"),
		Todos:       filepath.Join(tmpDir, "todos"),
		FileHistory: filepath.Join(tmpDir, "file-history"),
		SessionEnv:  filepath.Join(tmpDir, "session-env"),
	}

	// Create projects directory with one empty and one non-empty session
	projectDir := filepath.Join(paths.Projects, "-test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Empty session file (0 bytes)
	emptySession := filepath.Join(projectDir, "empty.jsonl")
	require.NoError(t, os.WriteFile(emptySession, []byte{}, 0644))

	// Non-empty session file
	validSession := filepath.Join(projectDir, "valid.jsonl")
	require.NoError(t, os.WriteFile(validSession, []byte(`{"sessionId":"sess1","cwd":"/test"}`), 0644))

	orphans, err := FindOrphans(paths, []string{"sess1"})
	require.NoError(t, err)

	// Should find the empty session file
	var emptySessionOrphans []OrphanResult
	for _, o := range orphans {
		if o.Type == OrphanTypeEmptySession {
			emptySessionOrphans = append(emptySessionOrphans, o)
		}
	}
	assert.Len(t, emptySessionOrphans, 1)
	assert.Equal(t, emptySession, emptySessionOrphans[0].Path)
}

func TestFindOrphans_OrphanTodos(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &claude.Paths{
		Root:        tmpDir,
		Projects:    filepath.Join(tmpDir, "projects"),
		Todos:       filepath.Join(tmpDir, "todos"),
		FileHistory: filepath.Join(tmpDir, "file-history"),
		SessionEnv:  filepath.Join(tmpDir, "session-env"),
	}

	require.NoError(t, os.MkdirAll(paths.Todos, 0755))

	// Valid todo (session exists)
	validTodo := filepath.Join(paths.Todos, "sess1-agent-abc.json")
	require.NoError(t, os.WriteFile(validTodo, []byte(`{}`), 0644))

	// Orphan todo (session doesn't exist)
	orphanTodo := filepath.Join(paths.Todos, "orphan-sess-agent-xyz.json")
	require.NoError(t, os.WriteFile(orphanTodo, []byte(`{}`), 0644))

	orphans, err := FindOrphans(paths, []string{"sess1"})
	require.NoError(t, err)

	// Should find the orphan todo
	var todoOrphans []OrphanResult
	for _, o := range orphans {
		if o.Type == OrphanTypeTodo {
			todoOrphans = append(todoOrphans, o)
		}
	}
	assert.Len(t, todoOrphans, 1)
	assert.Equal(t, orphanTodo, todoOrphans[0].Path)
}

func TestFindOrphans_OrphanFileHistory(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &claude.Paths{
		Root:        tmpDir,
		Projects:    filepath.Join(tmpDir, "projects"),
		Todos:       filepath.Join(tmpDir, "todos"),
		FileHistory: filepath.Join(tmpDir, "file-history"),
		SessionEnv:  filepath.Join(tmpDir, "session-env"),
	}

	require.NoError(t, os.MkdirAll(paths.FileHistory, 0755))

	// Valid file-history (session exists)
	validHistory := filepath.Join(paths.FileHistory, "sess1")
	require.NoError(t, os.MkdirAll(validHistory, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(validHistory, "file.txt"), []byte("content"), 0644))

	// Orphan file-history (session doesn't exist)
	orphanHistory := filepath.Join(paths.FileHistory, "orphan-sess")
	require.NoError(t, os.MkdirAll(orphanHistory, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(orphanHistory, "file.txt"), []byte("content"), 0644))

	orphans, err := FindOrphans(paths, []string{"sess1"})
	require.NoError(t, err)

	// Should find the orphan file-history
	var historyOrphans []OrphanResult
	for _, o := range orphans {
		if o.Type == OrphanTypeFileHistory {
			historyOrphans = append(historyOrphans, o)
		}
	}
	assert.Len(t, historyOrphans, 1)
	assert.Equal(t, orphanHistory, historyOrphans[0].Path)
}

func TestFindOrphans_EmptySessionEnv(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &claude.Paths{
		Root:        tmpDir,
		Projects:    filepath.Join(tmpDir, "projects"),
		Todos:       filepath.Join(tmpDir, "todos"),
		FileHistory: filepath.Join(tmpDir, "file-history"),
		SessionEnv:  filepath.Join(tmpDir, "session-env"),
	}

	require.NoError(t, os.MkdirAll(paths.SessionEnv, 0755))

	// Non-empty session-env (has files)
	nonEmptyEnv := filepath.Join(paths.SessionEnv, "sess1")
	require.NoError(t, os.MkdirAll(nonEmptyEnv, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nonEmptyEnv, "env.txt"), []byte("content"), 0644))

	// Empty session-env directory
	emptyEnv := filepath.Join(paths.SessionEnv, "sess2")
	require.NoError(t, os.MkdirAll(emptyEnv, 0755))

	orphans, err := FindOrphans(paths, []string{"sess1", "sess2"})
	require.NoError(t, err)

	// Should find the empty session-env
	var envOrphans []OrphanResult
	for _, o := range orphans {
		if o.Type == OrphanTypeSessionEnv {
			envOrphans = append(envOrphans, o)
		}
	}
	assert.Len(t, envOrphans, 1)
	assert.Equal(t, emptyEnv, envOrphans[0].Path)
}

func TestFindOrphans_NoOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &claude.Paths{
		Root:        tmpDir,
		Projects:    filepath.Join(tmpDir, "projects"),
		Todos:       filepath.Join(tmpDir, "todos"),
		FileHistory: filepath.Join(tmpDir, "file-history"),
		SessionEnv:  filepath.Join(tmpDir, "session-env"),
	}

	// Create all directories but no orphan data
	require.NoError(t, os.MkdirAll(paths.Projects, 0755))
	require.NoError(t, os.MkdirAll(paths.Todos, 0755))
	require.NoError(t, os.MkdirAll(paths.FileHistory, 0755))
	require.NoError(t, os.MkdirAll(paths.SessionEnv, 0755))

	orphans, err := FindOrphans(paths, []string{"sess1"})
	require.NoError(t, err)
	assert.Empty(t, orphans)
}

func TestFindOrphans_MissingDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	paths := &claude.Paths{
		Root:        tmpDir,
		Projects:    filepath.Join(tmpDir, "projects"),
		Todos:       filepath.Join(tmpDir, "todos"),
		FileHistory: filepath.Join(tmpDir, "file-history"),
		SessionEnv:  filepath.Join(tmpDir, "session-env"),
	}

	// Don't create any directories - should handle gracefully
	orphans, err := FindOrphans(paths, []string{"sess1"})
	require.NoError(t, err)
	assert.Empty(t, orphans)
}

func TestCleanOrphans_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	orphanPath := filepath.Join(tmpDir, "orphan.json")
	require.NoError(t, os.WriteFile(orphanPath, []byte(`{}`), 0644))

	orphans := []OrphanResult{
		{
			Type:      OrphanTypeTodo,
			Path:      orphanPath,
			SizeSaved: 2,
		},
	}

	results, err := CleanOrphans(orphans, true)
	require.NoError(t, err)

	// Dry run should not delete
	assert.FileExists(t, orphanPath)
	assert.Len(t, results, 1)
	assert.Equal(t, int64(2), results[0].SizeSaved)
}

func TestCleanOrphans_ActualDelete(t *testing.T) {
	tmpDir := t.TempDir()

	// Create orphan file
	orphanFile := filepath.Join(tmpDir, "orphan.json")
	require.NoError(t, os.WriteFile(orphanFile, []byte(`{}`), 0644))

	// Create orphan directory
	orphanDir := filepath.Join(tmpDir, "orphan-dir")
	require.NoError(t, os.MkdirAll(orphanDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(orphanDir, "file.txt"), []byte("content"), 0644))

	orphans := []OrphanResult{
		{
			Type:      OrphanTypeTodo,
			Path:      orphanFile,
			SizeSaved: 2,
		},
		{
			Type:      OrphanTypeFileHistory,
			Path:      orphanDir,
			SizeSaved: 7,
		},
	}

	results, err := CleanOrphans(orphans, false)
	require.NoError(t, err)

	// Should have deleted both
	assert.NoFileExists(t, orphanFile)
	assert.NoDirExists(t, orphanDir)
	assert.Len(t, results, 2)
}

func TestCleanOrphans_NonexistentPath(t *testing.T) {
	orphans := []OrphanResult{
		{
			Type:      OrphanTypeTodo,
			Path:      "/nonexistent/path",
			SizeSaved: 0,
		},
	}

	// Should not error for nonexistent paths
	results, err := CleanOrphans(orphans, false)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, int64(0), results[0].SizeSaved)
}

func TestBuildOrphanPreview(t *testing.T) {
	orphans := []OrphanResult{
		{
			Type:      OrphanTypeEmptySession,
			Path:      "/projects/-test/empty.jsonl",
			SizeSaved: 0,
		},
		{
			Type:      OrphanTypeTodo,
			Path:      "/todos/orphan-agent.json",
			SizeSaved: 1024,
		},
		{
			Type:      OrphanTypeFileHistory,
			Path:      "/file-history/orphan-sess",
			SizeSaved: 1024 * 1024,
		},
		{
			Type:      OrphanTypeSessionEnv,
			Path:      "/session-env/empty-sess",
			SizeSaved: 0,
		},
	}

	preview := BuildOrphanPreview(orphans)

	assert.Equal(t, "Orphan Cleanup", preview.Title)
	assert.Len(t, preview.Changes, 4)

	// Verify each orphan type is represented
	types := make(map[string]bool)
	for _, c := range preview.Changes {
		types[c.Description] = true
	}
	assert.True(t, types["Empty session file"])
	assert.True(t, types["Orphan todo"])
	assert.True(t, types["Orphan file history"])
	assert.True(t, types["Empty session env"])
}

func TestBuildOrphanPreview_Empty(t *testing.T) {
	preview := BuildOrphanPreview(nil)

	assert.Equal(t, "Orphan Cleanup", preview.Title)
	assert.Empty(t, preview.Changes)
}

func TestOrphanResult_TotalSize(t *testing.T) {
	orphans := []OrphanResult{
		{SizeSaved: 100},
		{SizeSaved: 200},
		{SizeSaved: 300},
	}

	var total int64
	for _, o := range orphans {
		total += o.SizeSaved
	}
	assert.Equal(t, int64(600), total)
}

func TestExtractSessionIDFromTodoFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"sess1-agent-abc.json", "sess1"},
		{"session-uuid-agent-agent-uuid.json", "session-uuid"},
		{"abc123-agent-xyz789.json", "abc123"},
		{"noagentsuffix.json", ""}, // No "-agent-" separator
		{"invalid.txt", ""},        // Wrong extension
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			result := extractSessionIDFromTodoFilename(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}
