package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSettings_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{
		"permissions": {
			"allow": ["Bash(git add:*)", "Read(**)"],
			"deny": ["Bash(rm -rf:*)"],
			"ask": ["Write(**)"]
		}
	}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(content), 0644))

	settings, err := LoadSettings(settingsPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"Bash(git add:*)", "Read(**)"}, settings.Permissions.Allow)
	assert.Equal(t, []string{"Bash(rm -rf:*)"}, settings.Permissions.Deny)
	assert.Equal(t, []string{"Write(**)"}, settings.Permissions.Ask)
}

func TestLoadSettings_MissingFile(t *testing.T) {
	settings, err := LoadSettings("/nonexistent/settings.json")
	require.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Empty(t, settings.Permissions.Allow)
	assert.Empty(t, settings.Permissions.Deny)
	assert.Empty(t, settings.Permissions.Ask)
}

func TestLoadSettings_EmptyPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{
		"permissions": {
			"allow": [],
			"deny": [],
			"ask": []
		}
	}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(content), 0644))

	settings, err := LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Empty(t, settings.Permissions.Allow)
	assert.Empty(t, settings.Permissions.Deny)
	assert.Empty(t, settings.Permissions.Ask)
}

func TestLoadSettings_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{invalid json}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(content), 0644))

	_, err := LoadSettings(settingsPath)
	assert.Error(t, err)
}

func TestLoadSettings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	require.NoError(t, os.WriteFile(settingsPath, []byte(""), 0644))

	settings, err := LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.NotNil(t, settings)
}

func TestLoadSettings_PartialPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Only "allow" is present
	content := `{
		"permissions": {
			"allow": ["Bash(git:*)"]
		}
	}`
	require.NoError(t, os.WriteFile(settingsPath, []byte(content), 0644))

	settings, err := LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"Bash(git:*)"}, settings.Permissions.Allow)
	assert.Empty(t, settings.Permissions.Deny)
	assert.Empty(t, settings.Permissions.Ask)
}

func TestSettings_Diff_AllUnique(t *testing.T) {
	local := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(npm:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{"Write(**)"},
		},
	}

	global := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(git:*)"},
			Deny:  []string{},
			Ask:   []string{},
		},
	}

	diff := local.Diff(global)

	// All local entries are unique (not in global)
	assert.Equal(t, []string{"Bash(npm:*)", "Read(**)"}, diff.Permissions.Allow)
	assert.Equal(t, []string{"Bash(rm:*)"}, diff.Permissions.Deny)
	assert.Equal(t, []string{"Write(**)"}, diff.Permissions.Ask)
}

func TestSettings_Diff_AllDuplicate(t *testing.T) {
	local := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(git:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{},
		},
	}

	global := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(git:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{},
		},
	}

	diff := local.Diff(global)

	// All local entries are duplicates (exist in global)
	assert.Empty(t, diff.Permissions.Allow)
	assert.Empty(t, diff.Permissions.Deny)
	assert.Empty(t, diff.Permissions.Ask)
}

func TestSettings_Diff_PartialDuplicate(t *testing.T) {
	local := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(git:*)", "Bash(npm:*)", "Read(**)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{"Write(**)"},
		},
	}

	global := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(git:*)", "Read(**)"},
			Deny:  []string{},
			Ask:   []string{"Write(**)"},
		},
	}

	diff := local.Diff(global)

	// Only unique entries should remain
	assert.Equal(t, []string{"Bash(npm:*)"}, diff.Permissions.Allow)
	assert.Equal(t, []string{"Bash(rm:*)"}, diff.Permissions.Deny)
	assert.Empty(t, diff.Permissions.Ask)
}

func TestSettings_Diff_EmptyLocal(t *testing.T) {
	local := &Settings{
		Permissions: Permissions{
			Allow: []string{},
			Deny:  []string{},
			Ask:   []string{},
		},
	}

	global := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(git:*)"},
			Deny:  []string{},
			Ask:   []string{},
		},
	}

	diff := local.Diff(global)

	assert.Empty(t, diff.Permissions.Allow)
	assert.Empty(t, diff.Permissions.Deny)
	assert.Empty(t, diff.Permissions.Ask)
}

func TestSettings_Diff_EmptyGlobal(t *testing.T) {
	local := &Settings{
		Permissions: Permissions{
			Allow: []string{"Bash(npm:*)"},
			Deny:  []string{"Bash(rm:*)"},
			Ask:   []string{},
		},
	}

	global := &Settings{
		Permissions: Permissions{
			Allow: []string{},
			Deny:  []string{},
			Ask:   []string{},
		},
	}

	diff := local.Diff(global)

	// All local entries are unique when global is empty
	assert.Equal(t, []string{"Bash(npm:*)"}, diff.Permissions.Allow)
	assert.Equal(t, []string{"Bash(rm:*)"}, diff.Permissions.Deny)
	assert.Empty(t, diff.Permissions.Ask)
}

func TestSettings_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		settings *Settings
		expected bool
	}{
		{
			name: "all empty",
			settings: &Settings{
				Permissions: Permissions{
					Allow: []string{},
					Deny:  []string{},
					Ask:   []string{},
				},
			},
			expected: true,
		},
		{
			name: "nil slices",
			settings: &Settings{
				Permissions: Permissions{},
			},
			expected: true,
		},
		{
			name: "has allow",
			settings: &Settings{
				Permissions: Permissions{
					Allow: []string{"Bash(git:*)"},
				},
			},
			expected: false,
		},
		{
			name: "has deny",
			settings: &Settings{
				Permissions: Permissions{
					Deny: []string{"Bash(rm:*)"},
				},
			},
			expected: false,
		},
		{
			name: "has ask",
			settings: &Settings{
				Permissions: Permissions{
					Ask: []string{"Write(**)"},
				},
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.settings.IsEmpty())
		})
	}
}
