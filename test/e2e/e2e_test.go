//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// getClaudeHome returns the path to the test Claude home directory.
func getClaudeHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// getCCCBinary returns the path to the cccc binary.
func getCCCBinary() string {
	return "/app/cccc"
}

// TestE2E_ListProjects verifies that list command works correctly.
func TestE2E_ListProjects(t *testing.T) {
	cmd := exec.Command(getCCCBinary(), "list", "projects")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list projects failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show active projects as OK
	if !strings.Contains(outputStr, "[OK]") {
		t.Errorf("expected [OK] status for active projects, got: %s", outputStr)
	}

	// Should show stale projects as STALE
	if !strings.Contains(outputStr, "[STALE]") {
		t.Errorf("expected [STALE] status for deleted projects, got: %s", outputStr)
	}

	// Should list the active project paths
	if !strings.Contains(outputStr, "/home/testuser/Code/active-project") {
		t.Errorf("expected active-project in output, got: %s", outputStr)
	}

	// Should list the stale project paths
	if !strings.Contains(outputStr, "/home/testuser/Code/deleted-project") {
		t.Errorf("expected deleted-project in output, got: %s", outputStr)
	}
}

// TestE2E_ListProjectsStaleOnly verifies --stale-only flag.
func TestE2E_ListProjectsStaleOnly(t *testing.T) {
	cmd := exec.Command(getCCCBinary(), "list", "projects", "--stale-only")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list projects --stale-only failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should NOT show active projects
	if strings.Contains(outputStr, "[OK]") {
		t.Errorf("--stale-only should not show [OK] projects, got: %s", outputStr)
	}

	// Should show stale projects
	if !strings.Contains(outputStr, "[STALE]") {
		t.Errorf("expected [STALE] status in --stale-only output, got: %s", outputStr)
	}
}

// TestE2E_CleanProjects_DryRun verifies dry-run doesn't delete anything.
func TestE2E_CleanProjects_DryRun(t *testing.T) {
	claudeHome := getClaudeHome()
	staleProjectDir := filepath.Join(claudeHome, "projects", "-home-testuser-Code-deleted-project")

	// Verify stale project exists before dry-run
	if _, err := os.Stat(staleProjectDir); os.IsNotExist(err) {
		t.Fatalf("stale project directory should exist before test: %s", staleProjectDir)
	}

	cmd := exec.Command(getCCCBinary(), "clean", "projects", "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clean projects --dry-run failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show DRY RUN message
	if !strings.Contains(outputStr, "DRY RUN") {
		t.Errorf("expected DRY RUN in output, got: %s", outputStr)
	}

	// Should show preview
	if !strings.Contains(outputStr, "deleted-project") {
		t.Errorf("expected deleted-project in preview, got: %s", outputStr)
	}

	// Verify NOTHING was deleted
	if _, err := os.Stat(staleProjectDir); os.IsNotExist(err) {
		t.Errorf("dry-run should NOT delete project directory")
	}
}

// TestE2E_CleanProjects_Confirmation_No verifies that 'n' aborts.
func TestE2E_CleanProjects_Confirmation_No(t *testing.T) {
	claudeHome := getClaudeHome()
	staleProjectDir := filepath.Join(claudeHome, "projects", "-home-testuser-Code-deleted-project")

	// Verify stale project exists
	if _, err := os.Stat(staleProjectDir); os.IsNotExist(err) {
		t.Fatalf("stale project directory should exist before test: %s", staleProjectDir)
	}

	cmd := exec.Command(getCCCBinary(), "clean", "projects")
	cmd.Stdin = strings.NewReader("n\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clean projects with 'n' failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show aborted message
	if !strings.Contains(outputStr, "Aborted") {
		t.Errorf("expected 'Aborted' message, got: %s", outputStr)
	}

	// Verify NOTHING was deleted
	if _, err := os.Stat(staleProjectDir); os.IsNotExist(err) {
		t.Errorf("'n' response should NOT delete project directory")
	}
}

// TestE2E_CleanProjects verifies actual cleanup.
func TestE2E_CleanProjects(t *testing.T) {
	claudeHome := getClaudeHome()
	staleProjectDir := filepath.Join(claudeHome, "projects", "-home-testuser-Code-deleted-project")
	activeProjectDir := filepath.Join(claudeHome, "projects", "-home-testuser-Code-active-project")

	// Verify directories exist before cleanup
	if _, err := os.Stat(staleProjectDir); os.IsNotExist(err) {
		t.Fatalf("stale project directory should exist before test: %s", staleProjectDir)
	}
	if _, err := os.Stat(activeProjectDir); os.IsNotExist(err) {
		t.Fatalf("active project directory should exist before test: %s", activeProjectDir)
	}

	// Run cleanup with --yes
	cmd := exec.Command(getCCCBinary(), "clean", "projects", "--yes")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clean projects --yes failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show cleanup message
	if !strings.Contains(outputStr, "Cleaned") {
		t.Errorf("expected 'Cleaned' message, got: %s", outputStr)
	}

	// Verify stale projects were deleted
	if _, err := os.Stat(staleProjectDir); !os.IsNotExist(err) {
		t.Errorf("stale project directory should be deleted: %s", staleProjectDir)
	}

	// Verify active projects were kept
	if _, err := os.Stat(activeProjectDir); os.IsNotExist(err) {
		t.Errorf("active project directory should NOT be deleted: %s", activeProjectDir)
	}

	// Verify audit log was written
	auditLog := filepath.Join(claudeHome, "cccc-audit.log")
	if _, err := os.Stat(auditLog); os.IsNotExist(err) {
		t.Errorf("audit log should be created: %s", auditLog)
	}

	auditContent, err := os.ReadFile(auditLog)
	if err != nil {
		t.Errorf("failed to read audit log: %v", err)
	}

	if !strings.Contains(string(auditContent), "DELETE") {
		t.Errorf("audit log should contain DELETE entries")
	}
}

// TestE2E_ListOrphans verifies orphan listing.
func TestE2E_ListOrphans(t *testing.T) {
	cmd := exec.Command(getCCCBinary(), "list", "orphans")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list orphans failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// After cleaning projects, there should be orphans (todos/file-history from deleted sessions)
	// The output depends on what was cleaned in previous tests
	t.Logf("Orphans output: %s", outputStr)
}

// TestE2E_Help verifies help command works.
func TestE2E_Help(t *testing.T) {
	cmd := exec.Command(getCCCBinary(), "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)

	// Should show usage info
	if !strings.Contains(outputStr, "cccc") {
		t.Errorf("expected 'cccc' in help output, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "clean") {
		t.Errorf("expected 'clean' command in help, got: %s", outputStr)
	}

	if !strings.Contains(outputStr, "list") {
		t.Errorf("expected 'list' command in help, got: %s", outputStr)
	}
}
