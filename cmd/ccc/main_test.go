package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setTestHome sets the home directory for tests.
// On Windows, os.UserHomeDir() uses USERPROFILE, not HOME.
func setTestHome(t *testing.T, tmpDir string) func() {
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")

	os.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tmpDir)
	}

	return func() {
		os.Setenv("HOME", oldHome)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", oldUserProfile)
		}
	}
}

func TestParseArgs_NoArgs(t *testing.T) {
	args, err := parseArgs([]string{})
	require.NoError(t, err)
	assert.Equal(t, "", args.Command)
	assert.True(t, args.Help)
}

func TestParseArgs_Help(t *testing.T) {
	for _, arg := range []string{"-h", "--help", "help"} {
		args, err := parseArgs([]string{arg})
		require.NoError(t, err)
		assert.True(t, args.Help, "expected help for %s", arg)
	}
}

func TestParseArgs_CleanCommand(t *testing.T) {
	args, err := parseArgs([]string{"clean"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.Equal(t, "", args.Subcommand)
	assert.False(t, args.DryRun)
	assert.False(t, args.Yes)
}

func TestParseArgs_CleanProjects(t *testing.T) {
	args, err := parseArgs([]string{"clean", "projects"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.Equal(t, "projects", args.Subcommand)
}

func TestParseArgs_CleanOrphans(t *testing.T) {
	args, err := parseArgs([]string{"clean", "orphans"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.Equal(t, "orphans", args.Subcommand)
}

func TestParseArgs_CleanConfig(t *testing.T) {
	args, err := parseArgs([]string{"clean", "config"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.Equal(t, "config", args.Subcommand)
}

func TestParseArgs_CleanWithDryRun(t *testing.T) {
	args, err := parseArgs([]string{"clean", "--dry-run"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.True(t, args.DryRun)
}

func TestParseArgs_CleanWithYes(t *testing.T) {
	args, err := parseArgs([]string{"clean", "--yes"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.True(t, args.Yes)
}

func TestParseArgs_CleanWithShortYes(t *testing.T) {
	args, err := parseArgs([]string{"clean", "-y"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.True(t, args.Yes)
}

func TestParseArgs_CleanWithAllFlags(t *testing.T) {
	args, err := parseArgs([]string{"clean", "projects", "--dry-run", "--yes"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.Equal(t, "projects", args.Subcommand)
	assert.True(t, args.DryRun)
	assert.True(t, args.Yes)
}

func TestParseArgs_ListCommand(t *testing.T) {
	args, err := parseArgs([]string{"list"})
	require.NoError(t, err)
	assert.Equal(t, "list", args.Command)
}

func TestParseArgs_ListProjects(t *testing.T) {
	args, err := parseArgs([]string{"list", "projects"})
	require.NoError(t, err)
	assert.Equal(t, "list", args.Command)
	assert.Equal(t, "projects", args.Subcommand)
}

func TestParseArgs_ListProjectsStaleOnly(t *testing.T) {
	args, err := parseArgs([]string{"list", "projects", "--stale-only"})
	require.NoError(t, err)
	assert.Equal(t, "list", args.Command)
	assert.Equal(t, "projects", args.Subcommand)
	assert.True(t, args.StaleOnly)
}

func TestParseArgs_ListOrphans(t *testing.T) {
	args, err := parseArgs([]string{"list", "orphans"})
	require.NoError(t, err)
	assert.Equal(t, "list", args.Command)
	assert.Equal(t, "orphans", args.Subcommand)
}

func TestParseArgs_UnknownCommand(t *testing.T) {
	_, err := parseArgs([]string{"unknown"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestRunCLI_Help(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "ccc - CleanClaudeConfig")
	assert.Contains(t, stdout.String(), "Usage:")
}

func TestRunCLI_HelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"--help"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "ccc - CleanClaudeConfig")
}

func TestRunCLI_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"unknown"}, stdin, &stdout, &stderr)

	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "unknown command")
}

func TestRunCLI_ListProjectsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"list", "projects"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "No projects found")
}

func TestRunCLI_ListProjectsWithData(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a project directory with a session file
	projectDir := filepath.Join(projectsDir, "-test-project")
	require.NoError(t, os.MkdirAll(projectDir, 0755))

	// Create session file with cwd pointing to an existing directory
	existingDir := filepath.Join(tmpDir, "existing-project")
	require.NoError(t, os.MkdirAll(existingDir, 0755))

	// Use filepath.ToSlash for JSON to avoid Windows backslash escaping issues
	sessionData := `{"sessionId":"sess1","cwd":"` + filepath.ToSlash(existingDir) + `","timestamp":"2025-01-01T00:00:00Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(sessionData), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"list", "projects"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	// On Windows the output may have backslashes, so check for the base name
	assert.Contains(t, stdout.String(), "existing-project")
}

func TestRunCLI_CleanProjectsDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a stale project (cwd doesn't exist)
	projectDir := filepath.Join(projectsDir, "-nonexistent-path")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	// Use a path that doesn't exist on any platform
	nonexistentPath := filepath.Join(tmpDir, "this-path-does-not-exist-anywhere")
	sessionData := `{"sessionId":"sess1","cwd":"` + filepath.ToSlash(nonexistentPath) + `","timestamp":"2025-01-01T00:00:00Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(sessionData), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"clean", "projects", "--dry-run"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	// Project should still exist (dry run)
	assert.DirExists(t, projectDir)
}

func TestRunCLI_CleanProjectsWithConfirmation(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a stale project (cwd doesn't exist)
	projectDir := filepath.Join(projectsDir, "-nonexistent-path")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	// Use a path that doesn't exist on any platform
	nonexistentPath := filepath.Join(tmpDir, "this-path-does-not-exist-anywhere")
	sessionData := `{"sessionId":"sess1","cwd":"` + filepath.ToSlash(nonexistentPath) + `","timestamp":"2025-01-01T00:00:00Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(sessionData), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("y\n") // Confirm yes

	code := runCLI([]string{"clean", "projects"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	// Project should be deleted
	assert.NoDirExists(t, projectDir)
}

func TestRunCLI_CleanProjectsDeclined(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a stale project (cwd doesn't exist)
	projectDir := filepath.Join(projectsDir, "-nonexistent-path")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	// Use a path that doesn't exist on any platform
	nonexistentPath := filepath.Join(tmpDir, "this-path-does-not-exist-anywhere")
	sessionData := `{"sessionId":"sess1","cwd":"` + filepath.ToSlash(nonexistentPath) + `","timestamp":"2025-01-01T00:00:00Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(sessionData), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("n\n") // Decline

	code := runCLI([]string{"clean", "projects"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	// Project should still exist (declined)
	assert.DirExists(t, projectDir)
	assert.Contains(t, stdout.String(), "Aborted")
}

func TestRunCLI_CleanProjectsYesFlag(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")

	// Create a stale project (cwd doesn't exist)
	projectDir := filepath.Join(projectsDir, "-nonexistent-path")
	require.NoError(t, os.MkdirAll(projectDir, 0755))
	// Use a path that doesn't exist on any platform
	nonexistentPath := filepath.Join(tmpDir, "this-path-does-not-exist-anywhere")
	sessionData := `{"sessionId":"sess1","cwd":"` + filepath.ToSlash(nonexistentPath) + `","timestamp":"2025-01-01T00:00:00Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, "session.jsonl"), []byte(sessionData), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("") // No input needed with --yes

	code := runCLI([]string{"clean", "projects", "--yes"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	// Project should be deleted (auto-confirmed)
	assert.NoDirExists(t, projectDir)
}

func TestRunCLI_ListOrphans(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")
	todosDir := filepath.Join(claudeDir, "todos")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))
	require.NoError(t, os.MkdirAll(todosDir, 0755))

	// Create an orphan todo
	require.NoError(t, os.WriteFile(filepath.Join(todosDir, "orphan-agent-xyz.json"), []byte(`{}`), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"list", "orphans"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
}

func TestRunCLI_CleanOrphansDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")
	todosDir := filepath.Join(claudeDir, "todos")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))
	require.NoError(t, os.MkdirAll(todosDir, 0755))

	// Create an orphan todo
	orphanTodo := filepath.Join(todosDir, "orphan-agent-xyz.json")
	require.NoError(t, os.WriteFile(orphanTodo, []byte(`{}`), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"clean", "orphans", "--dry-run"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	// Todo should still exist (dry run)
	assert.FileExists(t, orphanTodo)
}

func TestParseArgs_VerboseFlag(t *testing.T) {
	args, err := parseArgs([]string{"clean", "config", "--verbose"})
	require.NoError(t, err)
	assert.Equal(t, "clean", args.Command)
	assert.Equal(t, "config", args.Subcommand)
	assert.True(t, args.Verbose)
}

func TestParseArgs_ShortVerboseFlag(t *testing.T) {
	args, err := parseArgs([]string{"clean", "config", "-v"})
	require.NoError(t, err)
	assert.True(t, args.Verbose)
}

func TestRunCLI_CleanConfigVerboseDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude")
	projectsDir := filepath.Join(claudeDir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	// Create global settings with some permissions (settings.json is the global config)
	globalSettings := `{"permissions":{"allow":["Bash(git:*)","Read(**)"],"deny":["Bash(rm -rf:*)"]}}`
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(globalSettings), 0644))

	// Create a project directory with local settings that duplicate global
	// Note: Local configs are named settings.local.json
	projectDir := filepath.Join(tmpDir, "myproject")
	projectClaudeDir := filepath.Join(projectDir, ".claude")
	require.NoError(t, os.MkdirAll(projectClaudeDir, 0755))
	localSettings := `{"permissions":{"allow":["Bash(git:*)","Bash(npm:*)"],"deny":["Bash(rm -rf:*)"]}}`
	require.NoError(t, os.WriteFile(filepath.Join(projectClaudeDir, "settings.local.json"), []byte(localSettings), 0644))

	// Register this project in ~/.claude/projects/ so ScanProjects can find it
	encodedProjectDir := filepath.Join(projectsDir, "-myproject")
	require.NoError(t, os.MkdirAll(encodedProjectDir, 0755))
	sessionData := `{"sessionId":"sess1","cwd":"` + filepath.ToSlash(projectDir) + `","timestamp":"2025-01-01T00:00:00Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(encodedProjectDir, "session.jsonl"), []byte(sessionData), 0644))

	// Set environment to use temp dir
	cleanup := setTestHome(t, tmpDir)
	defer cleanup()

	var stdout, stderr bytes.Buffer
	stdin := strings.NewReader("")

	code := runCLI([]string{"clean", "config", "--dry-run", "--verbose"}, stdin, &stdout, &stderr)

	assert.Equal(t, 0, code)
	output := stdout.String()
	// Verbose should show the specific duplicate entries
	assert.Contains(t, output, "Bash(git:*)")
	assert.Contains(t, output, "Bash(rm -rf:*)")
	// Should show the global config path
	assert.Contains(t, output, "settings.json")
}
