package cleaner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkoepf/cccc/internal/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindStaleProjects_AllStale(t *testing.T) {
	projects := []claude.Project{
		{EncodedName: "project1", ActualPath: "/nonexistent/path1"},
		{EncodedName: "project2", ActualPath: "/nonexistent/path2"},
	}

	stale := FindStaleProjects(projects)
	assert.Len(t, stale, 2)
}

func TestFindStaleProjects_NoneStale(t *testing.T) {
	tmpDir := t.TempDir()
	path1 := filepath.Join(tmpDir, "project1")
	path2 := filepath.Join(tmpDir, "project2")
	require.NoError(t, os.MkdirAll(path1, 0755))
	require.NoError(t, os.MkdirAll(path2, 0755))

	projects := []claude.Project{
		{EncodedName: "project1", ActualPath: path1},
		{EncodedName: "project2", ActualPath: path2},
	}

	stale := FindStaleProjects(projects)
	assert.Len(t, stale, 0)
}

func TestFindStaleProjects_Mixed(t *testing.T) {
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "existing")
	require.NoError(t, os.MkdirAll(existingPath, 0755))

	projects := []claude.Project{
		{EncodedName: "existing", ActualPath: existingPath},
		{EncodedName: "deleted", ActualPath: "/nonexistent/deleted"},
	}

	stale := FindStaleProjects(projects)
	assert.Len(t, stale, 1)
	assert.Equal(t, "deleted", stale[0].EncodedName)
}

func TestFindStaleProjects_EmptyActualPath(t *testing.T) {
	// Projects with empty ActualPath (couldn't determine cwd) are considered stale
	projects := []claude.Project{
		{EncodedName: "project1", ActualPath: ""},
	}

	stale := FindStaleProjects(projects)
	assert.Len(t, stale, 1)
}

func TestCleanStaleProject_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	projectDir := filepath.Join(projectsDir, "-test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Create a session file
	sessionFile := filepath.Join(projectDir, "session.jsonl")
	require.NoError(t, os.WriteFile(sessionFile, []byte(`{"cwd":"/nonexistent"}`), 0644))

	project := claude.Project{
		EncodedName: "-test-project",
		ActualPath:  "/nonexistent",
		TotalSize:   100,
		FileCount:   1,
	}

	result, err := CleanStaleProject(projectsDir, project, true)
	require.NoError(t, err)

	// Dry run should not delete
	assert.DirExists(t, projectDir)
	assert.Equal(t, int64(100), result.SizeSaved)
	assert.Equal(t, 1, result.FilesRemoved)
	assert.Equal(t, project.EncodedName, result.Project.EncodedName)
}

func TestCleanStaleProject_ActualDelete(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	projectDir := filepath.Join(projectsDir, "-test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Create session files
	sessionFile1 := filepath.Join(projectDir, "session1.jsonl")
	sessionFile2 := filepath.Join(projectDir, "session2.jsonl")
	require.NoError(t, os.WriteFile(sessionFile1, []byte(`{"cwd":"/nonexistent"}`), 0644))
	require.NoError(t, os.WriteFile(sessionFile2, []byte(`{"cwd":"/nonexistent"}`), 0644))

	project := claude.Project{
		EncodedName: "-test-project",
		ActualPath:  "/nonexistent",
		TotalSize:   200,
		FileCount:   2,
	}

	result, err := CleanStaleProject(projectsDir, project, false)
	require.NoError(t, err)

	// Should have deleted the directory
	assert.NoDirExists(t, projectDir)
	assert.Equal(t, int64(200), result.SizeSaved)
	assert.Equal(t, 2, result.FilesRemoved)
}

func TestCleanStaleProject_NonexistentProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	project := claude.Project{
		EncodedName: "-nonexistent",
		ActualPath:  "/nonexistent",
		TotalSize:   0,
		FileCount:   0,
	}

	// Should not error if project directory doesn't exist
	result, err := CleanStaleProject(projectsDir, project, false)
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.SizeSaved)
}

func TestBuildStalePreview(t *testing.T) {
	tmpDir := t.TempDir()
	existingPath := filepath.Join(tmpDir, "existing")
	require.NoError(t, os.MkdirAll(existingPath, 0755))

	staleProjects := []claude.Project{
		{
			EncodedName: "-deleted-project",
			ActualPath:  "/deleted/project",
			TotalSize:   1024 * 1024,
			FileCount:   5,
			LastUsed:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	keptProjects := []claude.Project{
		{
			EncodedName: "-existing-project",
			ActualPath:  existingPath,
			TotalSize:   512 * 1024,
			FileCount:   3,
			LastUsed:    time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	preview := BuildStalePreview(staleProjects, keptProjects)

	assert.Equal(t, "Stale Project Cleanup", preview.Title)
	assert.Len(t, preview.Changes, 1)
	assert.Len(t, preview.Kept, 1)

	change := preview.Changes[0]
	assert.Equal(t, "/deleted/project", change.Path)
	assert.Equal(t, int64(1024*1024), change.Size)
	assert.Contains(t, change.Description, "5 files")

	kept := preview.Kept[0]
	assert.Equal(t, existingPath, kept.Path)
}

func TestBuildStalePreview_NoStale(t *testing.T) {
	preview := BuildStalePreview(nil, nil)

	assert.Equal(t, "Stale Project Cleanup", preview.Title)
	assert.Len(t, preview.Changes, 0)
	assert.Len(t, preview.Kept, 0)
}
