package claude

import (
	"os"
	"path/filepath"
)

// Paths contains the standard Claude Code directory paths.
type Paths struct {
	Root        string // ~/.claude
	Projects    string // ~/.claude/projects
	Todos       string // ~/.claude/todos
	FileHistory string // ~/.claude/file-history
	SessionEnv  string // ~/.claude/session-env
	Settings    string // ~/.claude/settings.json
}

// DiscoverPaths returns the Claude Code paths for the current user.
// If claudeHome is empty, it uses the default ~/.claude location.
func DiscoverPaths(claudeHome string) (*Paths, error) {
	root := claudeHome
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		root = filepath.Join(home, ".claude")
	}

	return &Paths{
		Root:        root,
		Projects:    filepath.Join(root, "projects"),
		Todos:       filepath.Join(root, "todos"),
		FileHistory: filepath.Join(root, "file-history"),
		SessionEnv:  filepath.Join(root, "session-env"),
		Settings:    filepath.Join(root, "settings.json"),
	}, nil
}
