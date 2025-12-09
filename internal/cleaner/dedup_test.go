package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mhk/ccc/internal/claude"
	"github.com/mhk/ccc/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindLocalConfigsFromProjects_Fast(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project directories with local configs
	// Note: Local configs are named settings.local.json, not settings.json
	project1 := filepath.Join(tmpDir, "project1")
	project2 := filepath.Join(tmpDir, "project2")
	project3 := filepath.Join(tmpDir, "project3") // No config

	require.NoError(t, os.MkdirAll(filepath.Join(project1, ".claude"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(project2, ".claude"), 0755))
	require.NoError(t, os.MkdirAll(project3, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(project1, ".claude", "settings.local.json"), []byte(`{}`), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(project2, ".claude", "settings.local.json"), []byte(`{}`), 0644))

	// Test the fast method that only checks specific project directories
	projectPaths := []string{project1, project2, project3, "/nonexistent/path"}
	configs := FindLocalConfigsFromProjects(projectPaths, "")

	assert.Len(t, configs, 2)
	assert.Contains(t, configs, filepath.Join(project1, ".claude", "settings.local.json"))
	assert.Contains(t, configs, filepath.Join(project2, ".claude", "settings.local.json"))
}

func TestFindLocalConfigsFromProjects_ExcludesGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate home directory being registered as a project
	// This happens when user runs Claude Code from their home directory
	homeDir := tmpDir
	globalClaudeDir := filepath.Join(homeDir, ".claude")
	require.NoError(t, os.MkdirAll(globalClaudeDir, 0755))
	// Global config is settings.json
	globalSettings := filepath.Join(globalClaudeDir, "settings.json")
	require.NoError(t, os.WriteFile(globalSettings, []byte(`{}`), 0644))
	// Home dir might also have a settings.local.json (should be excluded too)
	homeLocalSettings := filepath.Join(globalClaudeDir, "settings.local.json")
	require.NoError(t, os.WriteFile(homeLocalSettings, []byte(`{}`), 0644))

	// Create a normal project with local config
	projectDir := filepath.Join(tmpDir, "myproject")
	projectClaudeDir := filepath.Join(projectDir, ".claude")
	require.NoError(t, os.MkdirAll(projectClaudeDir, 0755))
	localSettings := filepath.Join(projectClaudeDir, "settings.local.json")
	require.NoError(t, os.WriteFile(localSettings, []byte(`{}`), 0644))

	// Both home dir and project dir are in project paths
	// Exclude the home directory's local settings (it's tied to global)
	projectPaths := []string{homeDir, projectDir}
	configs := FindLocalConfigsFromProjects(projectPaths, homeLocalSettings)

	// Should only find the project config, not the home dir's local config
	assert.Len(t, configs, 1)
	assert.Equal(t, localSettings, configs[0])
}

func TestDeduplicateConfig_AllDuplicate(t *testing.T) {
	global := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(git:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{"Write(**)"},
		},
	}

	local := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(git:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{"Write(**)"},
		},
	}

	result := DeduplicateConfig("/path/to/local/settings.json", global, local)

	// All entries are duplicates
	assert.Equal(t, []string{"Bash(git:*)", "Read(**)"}, result.DuplicateAllow)
	assert.Equal(t, []string{"Bash(rm:*)"}, result.DuplicateDeny)
	assert.Equal(t, []string{"Write(**)"}, result.DuplicateAsk)
	assert.True(t, result.SuggestDelete)
}

func TestDeduplicateConfig_NoDuplicates(t *testing.T) {
	global := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(git:*)"},
		},
	}

	local := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(npm:*)"},
			Deny:  []string{"Bash(rm:*)"},
		},
	}

	result := DeduplicateConfig("/path/to/local/settings.json", global, local)

	assert.Empty(t, result.DuplicateAllow)
	assert.Empty(t, result.DuplicateDeny)
	assert.Empty(t, result.DuplicateAsk)
	assert.False(t, result.SuggestDelete)
}

func TestDeduplicateConfig_PartialDuplicate(t *testing.T) {
	global := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(git:*)", "Read(**)"},
		},
	}

	local := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(git:*)", "Bash(npm:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
		},
	}

	result := DeduplicateConfig("/path/to/local/settings.json", global, local)

	// Bash(git:*) and Read(**) are duplicates, Bash(npm:*) is unique
	assert.Equal(t, []string{"Bash(git:*)", "Read(**)"}, result.DuplicateAllow)
	assert.Empty(t, result.DuplicateDeny) // Bash(rm:*) is unique (not in global)
	assert.Empty(t, result.DuplicateAsk)
	assert.False(t, result.SuggestDelete) // Still has unique entries
}

func TestDeduplicateConfig_EmptyLocal(t *testing.T) {
	global := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(git:*)"},
		},
	}

	local := &claude.Settings{}

	result := DeduplicateConfig("/path/to/local/settings.json", global, local)

	assert.Empty(t, result.DuplicateAllow)
	assert.Empty(t, result.DuplicateDeny)
	assert.Empty(t, result.DuplicateAsk)
	assert.True(t, result.SuggestDelete) // Empty local should suggest deletion
}

func TestDeduplicateConfig_EmptyGlobal(t *testing.T) {
	global := &claude.Settings{}

	local := &claude.Settings{
		Permissions: claude.Permissions{
			Allow: []string{"Bash(npm:*)"},
		},
	}

	result := DeduplicateConfig("/path/to/local/settings.json", global, local)

	// No duplicates since global is empty
	assert.Empty(t, result.DuplicateAllow)
	assert.False(t, result.SuggestDelete)
}

func TestApplyDedup_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	content := `{"permissions":{"allow":["Bash(git:*)","Bash(npm:*)"],"deny":["Bash(rm:*)"]}}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(content), 0644))

	result := &DedupResult{
		LocalPath:      settingsPath,
		DuplicateAllow: []string{"Bash(git:*)"},
		SuggestDelete:  false,
	}

	err := ApplyDedup(result, true)
	require.NoError(t, err)

	// File should be unchanged in dry run
	data, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestApplyDedup_RemoveDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	content := `{"permissions":{"allow":["Bash(git:*)","Bash(npm:*)"],"deny":["Bash(rm:*)"]}}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(content), 0644))

	result := &DedupResult{
		LocalPath:      settingsPath,
		DuplicateAllow: []string{"Bash(git:*)"},
		SuggestDelete:  false,
	}

	err := ApplyDedup(result, false)
	require.NoError(t, err)

	// File should have Bash(git:*) removed from allow
	settings, err := claude.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"Bash(npm:*)"}, settings.Permissions.Allow)
	assert.Equal(t, []string{"Bash(rm:*)"}, settings.Permissions.Deny)
}

func TestApplyDedup_DeleteFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")
	require.NoError(t, os.WriteFile(settingsPath, []byte(`{"permissions":{}}`), 0644))

	result := &DedupResult{
		LocalPath:     settingsPath,
		SuggestDelete: true,
	}

	err := ApplyDedup(result, false)
	require.NoError(t, err)

	// File should be deleted
	assert.NoFileExists(t, settingsPath)
}

func TestApplyDedup_NonexistentFile(t *testing.T) {
	result := &DedupResult{
		LocalPath:     "/nonexistent/settings.json",
		SuggestDelete: true,
	}

	// Should not error for nonexistent file
	err := ApplyDedup(result, false)
	require.NoError(t, err)
}

func TestBuildDedupPreview(t *testing.T) {
	results := []DedupResult{
		{
			LocalPath:      "/project1/.claude/settings.json",
			DuplicateAllow: []string{"Bash(git:*)"},
			DuplicateDeny:  []string{"Bash(rm:*)"},
			SuggestDelete:  false,
		},
		{
			LocalPath:     "/project2/.claude/settings.json",
			SuggestDelete: true,
		},
	}

	preview := BuildDedupPreview(results)

	assert.Equal(t, "Config Deduplication", preview.Title)
	assert.Len(t, preview.Changes, 2)
	assert.Equal(t, ui.ActionModify, preview.Changes[0].Action)
	assert.Equal(t, ui.ActionDelete, preview.Changes[1].Action)
}

func TestBuildDedupPreview_Empty(t *testing.T) {
	preview := BuildDedupPreview(nil)

	assert.Equal(t, "Config Deduplication", preview.Title)
	assert.Empty(t, preview.Changes)
}

func TestDedupResult_HasDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		result   DedupResult
		expected bool
	}{
		{
			name:     "no duplicates",
			result:   DedupResult{},
			expected: false,
		},
		{
			name: "has allow duplicates",
			result: DedupResult{
				DuplicateAllow: []string{"Bash(git:*)"},
			},
			expected: true,
		},
		{
			name: "has deny duplicates",
			result: DedupResult{
				DuplicateDeny: []string{"Bash(rm:*)"},
			},
			expected: true,
		},
		{
			name: "has ask duplicates",
			result: DedupResult{
				DuplicateAsk: []string{"Write(**)"},
			},
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.result.HasDuplicates())
		})
	}
}

func TestDedupResult_TotalDuplicates(t *testing.T) {
	result := DedupResult{
		DuplicateAllow: []string{"a", "b"},
		DuplicateDeny:  []string{"c"},
		DuplicateAsk:   []string{"d", "e", "f"},
	}

	assert.Equal(t, 6, result.TotalDuplicates())
}

func TestDedupResult_FormatAuditDetails(t *testing.T) {
	tests := []struct {
		name     string
		result   DedupResult
		expected string
	}{
		{
			name: "allow only",
			result: DedupResult{
				DuplicateAllow: []string{"Bash(git:*)", "Read(**)"},
			},
			expected: "removed allow: Bash(git:*), Read(**)",
		},
		{
			name: "deny only",
			result: DedupResult{
				DuplicateDeny: []string{"Bash(rm -rf:*)"},
			},
			expected: "removed deny: Bash(rm -rf:*)",
		},
		{
			name: "ask only",
			result: DedupResult{
				DuplicateAsk: []string{"Write(**)"},
			},
			expected: "removed ask: Write(**)",
		},
		{
			name: "all categories",
			result: DedupResult{
				DuplicateAllow: []string{"Bash(git:*)"},
				DuplicateDeny:  []string{"Bash(rm:*)"},
				DuplicateAsk:   []string{"Write(**)"},
			},
			expected: "removed allow: Bash(git:*); deny: Bash(rm:*); ask: Write(**)",
		},
		{
			name: "suggest delete",
			result: DedupResult{
				DuplicateAllow: []string{"Bash(git:*)"},
				SuggestDelete:  true,
			},
			expected: "deleted (all entries were duplicates)",
		},
		{
			name:     "no duplicates",
			result:   DedupResult{},
			expected: "no changes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.result.FormatAuditDetails())
		})
	}
}

func TestBuildDedupPreview_Verbose(t *testing.T) {
	globalPath := "/home/user/.claude/settings.json"
	results := []DedupResult{
		{
			LocalPath:      "/project1/.claude/settings.json",
			DuplicateAllow: []string{"Bash(git:*)"},
			DuplicateDeny:  []string{"Bash(rm:*)"},
			SuggestDelete:  false,
		},
		{
			LocalPath:      "/project2/.claude/settings.json",
			DuplicateAllow: []string{"Read(**)"},
			SuggestDelete:  true,
		},
	}

	preview := BuildDedupPreviewVerbose(results, globalPath)

	assert.Equal(t, "Config Deduplication", preview.Title)
	assert.Len(t, preview.Changes, 2)

	// First change should have verbose description with duplicates listed
	assert.Contains(t, preview.Changes[0].Description, "Bash(git:*)")
	assert.Contains(t, preview.Changes[0].Description, "Bash(rm:*)")

	// Second change should indicate deletion
	assert.Equal(t, ui.ActionDelete, preview.Changes[1].Action)
	assert.Contains(t, preview.Changes[1].Description, "Read(**)")
}
