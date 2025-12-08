package cleaner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mhk/ccc/internal/claude"
	"github.com/mhk/ccc/internal/ui"
)

// StaleResult represents the result of cleaning a stale project.
type StaleResult struct {
	Project      claude.Project
	SizeSaved    int64
	FilesRemoved int
}

// FindStaleProjects returns projects whose ActualPath no longer exists on disk.
func FindStaleProjects(projects []claude.Project) []claude.Project {
	var stale []claude.Project
	for _, p := range projects {
		if !p.Exists() {
			stale = append(stale, p)
		}
	}
	return stale
}

// CleanStaleProject removes the session data directory for a stale project.
// If dryRun is true, it returns what would be deleted without making changes.
func CleanStaleProject(projectsDir string, project claude.Project, dryRun bool) (*StaleResult, error) {
	result := &StaleResult{
		Project:      project,
		SizeSaved:    project.TotalSize,
		FilesRemoved: project.FileCount,
	}

	projectPath := filepath.Join(projectsDir, project.EncodedName)

	// Check if the project directory exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		result.SizeSaved = 0
		result.FilesRemoved = 0
		return result, nil
	}

	if dryRun {
		return result, nil
	}

	// Actually delete the directory
	if err := os.RemoveAll(projectPath); err != nil {
		return nil, fmt.Errorf("failed to remove project directory %s: %w", projectPath, err)
	}

	return result, nil
}

// BuildStalePreview creates a preview of stale projects to be cleaned.
func BuildStalePreview(staleProjects, keptProjects []claude.Project) *ui.Preview {
	preview := &ui.Preview{
		Title: "Stale Project Cleanup",
	}

	for _, p := range staleProjects {
		description := fmt.Sprintf("%d files, last used: %s", p.FileCount, p.LastUsed.Format("2006-01-02"))
		if p.ActualPath == "" {
			description = fmt.Sprintf("%d files (no cwd found)", p.FileCount)
		}

		preview.Changes = append(preview.Changes, ui.Change{
			Action:      ui.ActionDelete,
			Path:        p.ActualPath,
			Description: description,
			Size:        p.TotalSize,
		})
	}

	for _, p := range keptProjects {
		description := fmt.Sprintf("%d files", p.FileCount)
		preview.Kept = append(preview.Kept, ui.Change{
			Path:        p.ActualPath,
			Description: description,
			Size:        p.TotalSize,
		})
	}

	return preview
}
