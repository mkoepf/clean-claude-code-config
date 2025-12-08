package claude

import (
	"os"
	"path/filepath"
	"time"
)

// Project represents a Claude Code project with its session data.
type Project struct {
	EncodedName string    // Directory name: -Users-mhk-Code-ccc
	ActualPath  string    // From cwd field: /Users/mhk/Code/ccc
	SessionIDs  []string  // UUIDs of sessions in this project
	TotalSize   int64     // Bytes used by session files
	LastUsed    time.Time // Most recent session timestamp
	FileCount   int       // Number of session files
}

// Exists checks if the project's actual path exists on disk.
func (p *Project) Exists() bool {
	if p.ActualPath == "" {
		return false
	}
	_, err := os.Stat(p.ActualPath)
	return err == nil
}

// ScanProjects scans the projects directory and returns information about each project.
func ScanProjects(projectsDir string) ([]Project, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := filepath.Join(projectsDir, entry.Name())
		project := Project{
			EncodedName: entry.Name(),
		}

		// Scan session files in the project directory
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
			info, err := ParseSessionFile(sessionPath)
			if err != nil {
				continue
			}

			project.FileCount++
			project.TotalSize += info.Size

			if !info.IsEmpty {
				if project.ActualPath == "" {
					project.ActualPath = info.CWD
				}
				if info.ID != "" {
					project.SessionIDs = append(project.SessionIDs, info.ID)
				}
				if info.Timestamp.After(project.LastUsed) {
					project.LastUsed = info.Timestamp
				}
			}
		}

		projects = append(projects, project)
	}

	return projects, nil
}
