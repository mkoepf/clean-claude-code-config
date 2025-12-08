package ui

import (
	"fmt"
	"io"
)

// Action represents the type of change.
type Action string

const (
	ActionDelete Action = "DELETE"
	ActionModify Action = "MODIFY"
	ActionCreate Action = "CREATE"
)

// Change represents a single change to be made.
type Change struct {
	Action      Action
	Path        string
	Description string
	Size        int64
}

// Preview represents a set of changes to be previewed and confirmed.
type Preview struct {
	Title   string
	Changes []Change
	Kept    []Change // Items that will NOT be changed (for context)
}

// TotalSize returns the total size of all changes.
func (p *Preview) TotalSize() int64 {
	var total int64
	for _, c := range p.Changes {
		total += c.Size
	}
	return total
}

// Display writes a formatted preview to the given writer.
func (p *Preview) Display(w io.Writer) error {
	fmt.Fprintf(w, "=== %s ===\n\n", p.Title)

	if len(p.Changes) > 0 {
		fmt.Fprintln(w, "Changes:")
		for i, c := range p.Changes {
			fmt.Fprintf(w, "  %d. [%s] %s\n", i+1, c.Action, c.Path)
			if c.Description != "" {
				fmt.Fprintf(w, "     %s\n", c.Description)
			}
			fmt.Fprintf(w, "     Size: %s\n", FormatSize(c.Size))
		}
		fmt.Fprintln(w)
	}

	if len(p.Kept) > 0 {
		fmt.Fprintln(w, "Kept (no changes):")
		for i, c := range p.Kept {
			fmt.Fprintf(w, "  %d. %s\n", i+1, c.Path)
			if c.Description != "" {
				fmt.Fprintf(w, "     %s\n", c.Description)
			}
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Total: %s\n", FormatSize(p.TotalSize()))
	return nil
}

// FormatSize formats a byte size as a human-readable string (e.g., "14 MB").
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
