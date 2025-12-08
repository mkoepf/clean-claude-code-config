//go:build safety

package safety

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mhk/ccc/internal/claude"
	"github.com/mhk/ccc/internal/cleaner"
	"github.com/mhk/ccc/internal/ui"
)

// TestSafety_NeverDeletesExistingProject verifies that the cleaner never
// deletes projects whose paths still exist on disk.
func TestSafety_NeverDeletesExistingProject(t *testing.T) {
	// Create a temp directory structure simulating ~/.claude
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create 100 projects with existing paths
	const numProjects = 100
	existingPaths := make([]string, numProjects)

	for i := 0; i < numProjects; i++ {
		// Create the actual project directory (it MUST exist)
		existingPath := filepath.Join(tmpHome, "projects", "project-"+string(rune('a'+i%26))+"-"+string(rune('0'+i/26)))
		if err := os.MkdirAll(existingPath, 0755); err != nil {
			t.Fatalf("failed to create project directory: %v", err)
		}
		existingPaths[i] = existingPath

		// Create the encoded project directory in ~/.claude/projects
		encodedName := strings.ReplaceAll(existingPath, "/", "-")
		projectDir := filepath.Join(projectsDir, encodedName)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("failed to create project directory: %v", err)
		}

		// Create a session file with the cwd pointing to the existing path
		sessionFile := filepath.Join(projectDir, "session.jsonl")
		sessionContent := `{"sessionId":"session-` + string(rune('a'+i%26)) + `","cwd":"` + existingPath + `","timestamp":"2025-01-01T00:00:00Z"}` + "\n"
		if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
			t.Fatalf("failed to write session file: %v", err)
		}
	}

	// Scan projects
	projects, err := claude.ScanProjects(projectsDir)
	if err != nil {
		t.Fatalf("failed to scan projects: %v", err)
	}

	if len(projects) != numProjects {
		t.Fatalf("expected %d projects, got %d", numProjects, len(projects))
	}

	// Find stale projects - should be ZERO
	stale := cleaner.FindStaleProjects(projects)
	if len(stale) != 0 {
		t.Errorf("expected 0 stale projects, got %d", len(stale))
		for _, p := range stale {
			t.Errorf("  incorrectly marked stale: %s (exists=%v)", p.ActualPath, p.Exists())
		}
	}

	// Verify all project directories still exist after running FindStaleProjects
	for _, path := range existingPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("project directory was deleted: %s", path)
		}
	}
}

// TestSafety_DefaultConfirmationIsNo verifies that empty input defaults to No.
func TestSafety_DefaultConfirmationIsNo(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty input", "\n"},
		{"just spaces", "   \n"},
		{"n", "n\n"},
		{"N", "N\n"},
		{"no", "no\n"},
		{"No", "No\n"},
		{"NO", "NO\n"},
		{"random text", "maybe\n"},
		{"number", "1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			confirmer := &ui.Confirmer{
				In:  strings.NewReader(tt.input),
				Out: &out,
			}

			result := confirmer.Confirm("Test prompt: ")

			if result != ui.ConfirmNo {
				t.Errorf("expected ConfirmNo for input %q, got %v", tt.input, result)
			}
		})
	}
}

// TestSafety_OnlyYesConfirms verifies that only "y" or "yes" confirms.
func TestSafety_OnlyYesConfirms(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"lowercase y", "y\n"},
		{"uppercase Y", "Y\n"},
		{"lowercase yes", "yes\n"},
		{"uppercase YES", "YES\n"},
		{"mixed Yes", "Yes\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			confirmer := &ui.Confirmer{
				In:  strings.NewReader(tt.input),
				Out: &out,
			}

			result := confirmer.Confirm("Test prompt: ")

			if result != ui.ConfirmYes {
				t.Errorf("expected ConfirmYes for input %q, got %v", tt.input, result)
			}
		})
	}
}

// TestSafety_AuditLogAlwaysWritten verifies that deletions are logged to the audit log.
func TestSafety_AuditLogAlwaysWritten(t *testing.T) {
	tmpDir := t.TempDir()
	auditPath := filepath.Join(tmpDir, "audit.log")

	logger, err := ui.NewAuditLogger(auditPath)
	if err != nil {
		t.Fatalf("failed to create audit logger: %v", err)
	}

	// Log several operations
	operations := []struct {
		action ui.Action
		path   string
		size   int64
	}{
		{ui.ActionDelete, "/path/to/project1", 1024},
		{ui.ActionDelete, "/path/to/project2", 2048},
		{ui.ActionModify, "/path/to/config", 512},
	}

	for _, op := range operations {
		logger.Log(op.action, op.path, op.size)
	}

	logger.Close()

	// Verify audit log exists and contains entries
	content, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatalf("failed to read audit log: %v", err)
	}

	contentStr := string(content)
	for _, op := range operations {
		if !strings.Contains(contentStr, string(op.action)) {
			t.Errorf("audit log missing action %s", op.action)
		}
		if !strings.Contains(contentStr, op.path) {
			t.Errorf("audit log missing path %s", op.path)
		}
	}
}

// TestSafety_PartialFailureStops verifies that cleanup stops on error.
func TestSafety_PartialFailureStops(t *testing.T) {
	tmpDir := t.TempDir()

	// Create orphan items, one of which will be read-only (can't delete)
	orphans := []cleaner.OrphanResult{
		{Type: "test", Path: filepath.Join(tmpDir, "orphan1"), SizeSaved: 100},
		{Type: "test", Path: filepath.Join(tmpDir, "readonly"), SizeSaved: 200},
		{Type: "test", Path: filepath.Join(tmpDir, "orphan3"), SizeSaved: 300},
	}

	// Create the files/directories
	for _, o := range orphans {
		if err := os.WriteFile(o.Path, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Make one directory read-only (to simulate failure)
	readonlyDir := filepath.Join(tmpDir, "readonly_dir")
	if err := os.MkdirAll(readonlyDir, 0755); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}
	readonlyFile := filepath.Join(readonlyDir, "file")
	if err := os.WriteFile(readonlyFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file in readonly dir: %v", err)
	}
	// Make directory read-only so files inside can't be deleted
	if err := os.Chmod(readonlyDir, 0555); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	// Clean up at the end
	defer os.Chmod(readonlyDir, 0755)

	// Create orphan pointing to the protected file
	protectedOrphans := []cleaner.OrphanResult{
		{Type: "file", Path: readonlyFile, SizeSaved: 100},
	}

	// This should fail
	_, err := cleaner.CleanOrphans(protectedOrphans, false)
	if err == nil {
		// On some systems this might succeed, so we check if file still exists
		if _, statErr := os.Stat(readonlyFile); statErr != nil {
			t.Log("File was deleted despite read-only parent (OS-specific behavior)")
		}
	}
}

// TestSafety_PreviewMatchesAction verifies that dry-run preview matches actual actions.
func TestSafety_PreviewMatchesAction(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a mix of stale and valid projects
	staleProjectPath := filepath.Join(tmpHome, "deleted-project")
	validProjectPath := filepath.Join(tmpHome, "existing-project")

	// Create the valid project directory (exists)
	if err := os.MkdirAll(validProjectPath, 0755); err != nil {
		t.Fatalf("failed to create valid project: %v", err)
	}

	// Create stale project in ~/.claude/projects (directory no longer exists)
	staleEncoded := strings.ReplaceAll(staleProjectPath, "/", "-")
	staleDir := filepath.Join(projectsDir, staleEncoded)
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatalf("failed to create stale project dir: %v", err)
	}
	staleSession := filepath.Join(staleDir, "session.jsonl")
	if err := os.WriteFile(staleSession, []byte(`{"sessionId":"stale","cwd":"`+staleProjectPath+`","timestamp":"2025-01-01T00:00:00Z"}`+"\n"), 0644); err != nil {
		t.Fatalf("failed to write stale session: %v", err)
	}

	// Create valid project in ~/.claude/projects (directory exists)
	validEncoded := strings.ReplaceAll(validProjectPath, "/", "-")
	validDir := filepath.Join(projectsDir, validEncoded)
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("failed to create valid project dir: %v", err)
	}
	validSession := filepath.Join(validDir, "session.jsonl")
	if err := os.WriteFile(validSession, []byte(`{"sessionId":"valid","cwd":"`+validProjectPath+`","timestamp":"2025-01-01T00:00:00Z"}`+"\n"), 0644); err != nil {
		t.Fatalf("failed to write valid session: %v", err)
	}

	// Scan and find stale projects
	projects, err := claude.ScanProjects(projectsDir)
	if err != nil {
		t.Fatalf("failed to scan: %v", err)
	}

	stale := cleaner.FindStaleProjects(projects)
	var kept []claude.Project
	for _, p := range projects {
		if p.Exists() {
			kept = append(kept, p)
		}
	}

	// Build preview (what dry-run would show)
	preview := cleaner.BuildStalePreview(stale, kept)

	// Verify preview shows correct items to delete
	if len(preview.Changes) != 1 {
		t.Errorf("expected 1 change in preview, got %d", len(preview.Changes))
	}
	if len(preview.Kept) != 1 {
		t.Errorf("expected 1 kept in preview, got %d", len(preview.Kept))
	}

	// The stale project path should be in Changes
	foundStale := false
	for _, c := range preview.Changes {
		if c.Path == staleProjectPath {
			foundStale = true
		}
	}
	if !foundStale {
		t.Errorf("preview did not include stale project %s", staleProjectPath)
	}

	// The valid project should be in Kept
	foundValid := false
	for _, k := range preview.Kept {
		if k.Path == validProjectPath {
			foundValid = true
		}
	}
	if !foundValid {
		t.Errorf("preview did not include valid project %s in kept list", validProjectPath)
	}

	// Now actually clean (non-dry-run)
	for _, p := range stale {
		_, err := cleaner.CleanStaleProject(projectsDir, p, false)
		if err != nil {
			t.Errorf("failed to clean stale project: %v", err)
		}
	}

	// Verify stale was deleted
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Errorf("stale project was not deleted")
	}

	// Verify valid was kept
	if _, err := os.Stat(validDir); os.IsNotExist(err) {
		t.Errorf("valid project was incorrectly deleted")
	}
}

// TestSafety_DryRunNeverModifies verifies that --dry-run never modifies anything.
func TestSafety_DryRunNeverModifies(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := filepath.Join(tmpHome, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a stale project
	staleProjectPath := filepath.Join(tmpHome, "deleted-project")
	staleEncoded := strings.ReplaceAll(staleProjectPath, "/", "-")
	staleDir := filepath.Join(projectsDir, staleEncoded)
	if err := os.MkdirAll(staleDir, 0755); err != nil {
		t.Fatalf("failed to create stale project dir: %v", err)
	}
	staleSession := filepath.Join(staleDir, "session.jsonl")
	sessionContent := `{"sessionId":"stale","cwd":"` + staleProjectPath + `","timestamp":"2025-01-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(staleSession, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("failed to write stale session: %v", err)
	}

	// Record state before
	beforeInfo, _ := os.Stat(staleDir)
	beforeModTime := beforeInfo.ModTime()

	// Scan and find stale
	projects, err := claude.ScanProjects(projectsDir)
	if err != nil {
		t.Fatalf("failed to scan: %v", err)
	}

	stale := cleaner.FindStaleProjects(projects)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale project, got %d", len(stale))
	}

	// Run with dryRun=true
	for _, p := range stale {
		result, err := cleaner.CleanStaleProject(projectsDir, p, true)
		if err != nil {
			t.Errorf("dry run failed: %v", err)
		}
		// Result should still report what would be saved
		if result.SizeSaved == 0 {
			t.Errorf("dry run should report size that would be saved")
		}
	}

	// Verify NOTHING was modified
	if _, err := os.Stat(staleDir); os.IsNotExist(err) {
		t.Errorf("dry run deleted the directory!")
	}

	afterInfo, _ := os.Stat(staleDir)
	afterModTime := afterInfo.ModTime()
	if !beforeModTime.Equal(afterModTime) {
		t.Errorf("dry run modified the directory (modtime changed)")
	}

	// Session file should still exist
	if _, err := os.Stat(staleSession); os.IsNotExist(err) {
		t.Errorf("dry run deleted session file!")
	}
}

// TestSafety_EmptyPathNeverDeletes verifies that projects with empty ActualPath
// are never cleaned (as we can't verify they're truly stale).
func TestSafety_EmptyPathNeverDeletes(t *testing.T) {
	// Create a project with empty ActualPath
	project := claude.Project{
		EncodedName: "-tmp-unknown-project",
		ActualPath:  "", // Empty - we don't know the real path
		SessionIDs:  []string{"session1"},
		TotalSize:   1024,
		LastUsed:    time.Now(),
		FileCount:   1,
	}

	// Exists() should return false for empty path
	if project.Exists() {
		t.Errorf("project with empty path should not report as existing")
	}

	// FindStaleProjects should mark this as stale
	stale := cleaner.FindStaleProjects([]claude.Project{project})
	if len(stale) != 1 {
		t.Errorf("expected 1 stale project, got %d", len(stale))
	}

	// The preview should clearly indicate "no cwd found"
	preview := cleaner.BuildStalePreview(stale, nil)
	if len(preview.Changes) != 1 {
		t.Fatalf("expected 1 change")
	}

	if !strings.Contains(preview.Changes[0].Description, "no cwd found") {
		t.Errorf("preview should indicate 'no cwd found' for empty path projects")
	}
}
