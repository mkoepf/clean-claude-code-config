package claude

import (
	"encoding/json"
	"os"
)

// Settings represents Claude Code settings configuration.
type Settings struct {
	Permissions Permissions `json:"permissions"`
}

// Permissions represents the permissions configuration.
type Permissions struct {
	Allow []string `json:"allow"`
	Deny  []string `json:"deny"`
	Ask   []string `json:"ask"`
}

// LoadSettings loads settings from the given path.
// Returns an empty Settings if the file doesn't exist.
func LoadSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Settings{}, nil
	}
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &Settings{}, nil
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// Diff returns a new Settings containing entries in s that are not in other.
func (s *Settings) Diff(other *Settings) *Settings {
	return &Settings{
		Permissions: Permissions{
			Allow: diffSlice(s.Permissions.Allow, other.Permissions.Allow),
			Deny:  diffSlice(s.Permissions.Deny, other.Permissions.Deny),
			Ask:   diffSlice(s.Permissions.Ask, other.Permissions.Ask),
		},
	}
}

// IsEmpty returns true if all permission lists are empty.
func (s *Settings) IsEmpty() bool {
	return len(s.Permissions.Allow) == 0 &&
		len(s.Permissions.Deny) == 0 &&
		len(s.Permissions.Ask) == 0
}

// diffSlice returns elements in a that are not in b.
func diffSlice(a, b []string) []string {
	if len(a) == 0 {
		return nil
	}

	bSet := make(map[string]struct{}, len(b))
	for _, v := range b {
		bSet[v] = struct{}{}
	}

	var result []string
	for _, v := range a {
		if _, exists := bSet[v]; !exists {
			result = append(result, v)
		}
	}

	return result
}
