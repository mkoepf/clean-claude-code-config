package cleaner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mhk/ccc/internal/claude"
	"github.com/mhk/ccc/internal/ui"
)

// OrphanType identifies the type of orphan data.
type OrphanType string

const (
	OrphanTypeEmptySession OrphanType = "empty_session"
	OrphanTypeTodo         OrphanType = "todo"
	OrphanTypeFileHistory  OrphanType = "file_history"
	OrphanTypeSessionEnv   OrphanType = "session_env"
)

// OrphanResult represents an orphan item found during scanning.
type OrphanResult struct {
	Type      OrphanType
	Path      string
	SizeSaved int64
}

// FindOrphans scans the Claude directories for orphan data.
// validSessionIDs is a list of session IDs that are still valid.
func FindOrphans(paths *claude.Paths, validSessionIDs []string) ([]OrphanResult, error) {
	validIDs := make(map[string]struct{}, len(validSessionIDs))
	for _, id := range validSessionIDs {
		validIDs[id] = struct{}{}
	}

	var orphans []OrphanResult

	// Find empty session files
	emptyOrphans, err := findEmptySessions(paths.Projects)
	if err != nil {
		return nil, err
	}
	orphans = append(orphans, emptyOrphans...)

	// Find orphan todos
	todoOrphans, err := findOrphanTodos(paths.Todos, validIDs)
	if err != nil {
		return nil, err
	}
	orphans = append(orphans, todoOrphans...)

	// Find orphan file-history
	historyOrphans, err := findOrphanFileHistory(paths.FileHistory, validIDs)
	if err != nil {
		return nil, err
	}
	orphans = append(orphans, historyOrphans...)

	// Find empty session-env directories
	envOrphans, err := findEmptySessionEnv(paths.SessionEnv)
	if err != nil {
		return nil, err
	}
	orphans = append(orphans, envOrphans...)

	return orphans, nil
}

// findEmptySessions finds 0-byte .jsonl files in the projects directory.
func findEmptySessions(projectsDir string) ([]OrphanResult, error) {
	var orphans []OrphanResult

	if _, err := os.Stat(projectsDir); os.IsNotExist(err) {
		return orphans, nil
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := filepath.Join(projectsDir, entry.Name())
		sessionEntries, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, sessionEntry := range sessionEntries {
			if sessionEntry.IsDir() {
				continue
			}
			if filepath.Ext(sessionEntry.Name()) != ".jsonl" {
				continue
			}

			sessionPath := filepath.Join(projectPath, sessionEntry.Name())
			info, err := sessionEntry.Info()
			if err != nil {
				continue
			}

			if info.Size() == 0 {
				orphans = append(orphans, OrphanResult{
					Type:      OrphanTypeEmptySession,
					Path:      sessionPath,
					SizeSaved: 0,
				})
			}
		}
	}

	return orphans, nil
}

// findOrphanTodos finds todo files that reference non-existent sessions.
// Todo files are named: {sessionID}-agent-{agentID}.json
func findOrphanTodos(todosDir string, validIDs map[string]struct{}) ([]OrphanResult, error) {
	var orphans []OrphanResult

	if _, err := os.Stat(todosDir); os.IsNotExist(err) {
		return orphans, nil
	}

	entries, err := os.ReadDir(todosDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sessionID := extractSessionIDFromTodoFilename(entry.Name())
		if sessionID == "" {
			continue
		}

		if _, exists := validIDs[sessionID]; !exists {
			todoPath := filepath.Join(todosDir, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			orphans = append(orphans, OrphanResult{
				Type:      OrphanTypeTodo,
				Path:      todoPath,
				SizeSaved: info.Size(),
			})
		}
	}

	return orphans, nil
}

// extractSessionIDFromTodoFilename extracts the session ID from a todo filename.
// Format: {sessionID}-agent-{agentID}.json
func extractSessionIDFromTodoFilename(filename string) string {
	if !strings.HasSuffix(filename, ".json") {
		return ""
	}

	// Remove .json suffix
	name := strings.TrimSuffix(filename, ".json")

	// Find "-agent-" separator
	idx := strings.Index(name, "-agent-")
	if idx == -1 {
		return ""
	}

	return name[:idx]
}

// findOrphanFileHistory finds file-history directories for non-existent sessions.
func findOrphanFileHistory(historyDir string, validIDs map[string]struct{}) ([]OrphanResult, error) {
	var orphans []OrphanResult

	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		return orphans, nil
	}

	entries, err := os.ReadDir(historyDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionID := entry.Name()
		if _, exists := validIDs[sessionID]; !exists {
			historyPath := filepath.Join(historyDir, sessionID)
			size, err := dirSize(historyPath)
			if err != nil {
				continue
			}

			orphans = append(orphans, OrphanResult{
				Type:      OrphanTypeFileHistory,
				Path:      historyPath,
				SizeSaved: size,
			})
		}
	}

	return orphans, nil
}

// findEmptySessionEnv finds empty directories in session-env.
func findEmptySessionEnv(sessionEnvDir string) ([]OrphanResult, error) {
	var orphans []OrphanResult

	if _, err := os.Stat(sessionEnvDir); os.IsNotExist(err) {
		return orphans, nil
	}

	entries, err := os.ReadDir(sessionEnvDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		envPath := filepath.Join(sessionEnvDir, entry.Name())
		empty, err := isDirEmpty(envPath)
		if err != nil {
			continue
		}

		if empty {
			orphans = append(orphans, OrphanResult{
				Type:      OrphanTypeSessionEnv,
				Path:      envPath,
				SizeSaved: 0,
			})
		}
	}

	return orphans, nil
}

// isDirEmpty returns true if the directory contains no files.
func isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// dirSize calculates the total size of a directory and its contents.
func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// CleanOrphans removes the orphan items.
// If dryRun is true, returns what would be deleted without making changes.
func CleanOrphans(orphans []OrphanResult, dryRun bool) ([]OrphanResult, error) {
	results := make([]OrphanResult, len(orphans))
	copy(results, orphans)

	if dryRun {
		return results, nil
	}

	for i := range results {
		path := results[i].Path

		// Check if path exists
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			results[i].SizeSaved = 0
			continue
		}
		if err != nil {
			return results, err
		}

		// Remove file or directory
		if info.IsDir() {
			if err := os.RemoveAll(path); err != nil {
				return results, err
			}
		} else {
			if err := os.Remove(path); err != nil {
				return results, err
			}
		}
	}

	return results, nil
}

// BuildOrphanPreview creates a preview of orphans to be cleaned.
func BuildOrphanPreview(orphans []OrphanResult) *ui.Preview {
	preview := &ui.Preview{
		Title: "Orphan Cleanup",
	}

	for _, o := range orphans {
		var description string
		switch o.Type {
		case OrphanTypeEmptySession:
			description = "Empty session file"
		case OrphanTypeTodo:
			description = "Orphan todo"
		case OrphanTypeFileHistory:
			description = "Orphan file history"
		case OrphanTypeSessionEnv:
			description = "Empty session env"
		}

		preview.Changes = append(preview.Changes, ui.Change{
			Action:      ui.ActionDelete,
			Path:        o.Path,
			Description: description,
			Size:        o.SizeSaved,
		})
	}

	return preview
}
