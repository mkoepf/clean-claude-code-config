package claude

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// SessionInfo contains metadata extracted from a session file.
type SessionInfo struct {
	ID        string
	CWD       string
	Timestamp time.Time
	FilePath  string
	Size      int64
	IsEmpty   bool
}

// sessionLine represents a single line from a session JSONL file.
type sessionLine struct {
	SessionID string    `json:"sessionId"`
	CWD       string    `json:"cwd"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrNoCWD is returned when no cwd field can be found in session files.
var ErrNoCWD = errors.New("no cwd field found in session files")

// ParseSessionFile reads a session JSONL file and extracts metadata.
func ParseSessionFile(path string) (*SessionInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	info := &SessionInfo{
		FilePath: path,
		Size:     stat.Size(),
	}

	if stat.Size() == 0 {
		info.IsEmpty = true
		return info, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var sl sessionLine
		if err := json.Unmarshal(line, &sl); err != nil {
			return nil, err
		}

		if sl.CWD != "" {
			info.ID = sl.SessionID
			info.CWD = sl.CWD
			info.Timestamp = sl.Timestamp
			return info, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, ErrNoCWD
}

// ExtractCWD reads the first valid line from session files in a project directory
// and returns the cwd field.
func ExtractCWD(projectDir string) (string, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}

		path := filepath.Join(projectDir, entry.Name())
		info, err := ParseSessionFile(path)
		if err != nil {
			continue
		}
		if info.IsEmpty {
			continue
		}
		if info.CWD != "" {
			return info.CWD, nil
		}
	}

	return "", ErrNoCWD
}
