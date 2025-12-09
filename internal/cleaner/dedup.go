package cleaner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mkoepf/cccc/internal/claude"
	"github.com/mkoepf/cccc/internal/ui"
)

// DedupResult represents the result of deduplicating a local config.
type DedupResult struct {
	LocalPath      string
	DuplicateAllow []string
	DuplicateDeny  []string
	DuplicateAsk   []string
	SuggestDelete  bool // True if local becomes empty after dedup
}

// HasDuplicates returns true if any duplicate entries were found.
func (r *DedupResult) HasDuplicates() bool {
	return len(r.DuplicateAllow) > 0 ||
		len(r.DuplicateDeny) > 0 ||
		len(r.DuplicateAsk) > 0
}

// TotalDuplicates returns the total number of duplicate entries found.
func (r *DedupResult) TotalDuplicates() int {
	return len(r.DuplicateAllow) + len(r.DuplicateDeny) + len(r.DuplicateAsk)
}

// FormatAuditDetails returns a human-readable description of the changes made.
func (r *DedupResult) FormatAuditDetails() string {
	if r.SuggestDelete {
		return "deleted (all entries were duplicates)"
	}

	if !r.HasDuplicates() {
		return "no changes"
	}

	var parts []string
	if len(r.DuplicateAllow) > 0 {
		parts = append(parts, "allow: "+strings.Join(r.DuplicateAllow, ", "))
	}
	if len(r.DuplicateDeny) > 0 {
		parts = append(parts, "deny: "+strings.Join(r.DuplicateDeny, ", "))
	}
	if len(r.DuplicateAsk) > 0 {
		parts = append(parts, "ask: "+strings.Join(r.DuplicateAsk, ", "))
	}

	return "removed " + strings.Join(parts, "; ")
}

// FindLocalConfigsFromProjects efficiently finds local .claude/settings.local.json files
// by only checking the specific project directories provided.
// It excludes the config file specified by excludePath (typically ~/.claude/settings.local.json).
// This is much faster than walking the entire home directory.
//
// Note: Local project configs are named "settings.local.json", not "settings.json".
// The global config at ~/.claude/settings.json is a different file.
func FindLocalConfigsFromProjects(projectPaths []string, excludePath string) []string {
	var configs []string

	// Normalize exclude path for comparison
	if excludePath != "" {
		excludePath = filepath.Clean(excludePath)
	}

	for _, projectPath := range projectPaths {
		// Local configs are named settings.local.json
		settingsPath := filepath.Join(projectPath, ".claude", "settings.local.json")
		if _, err := os.Stat(settingsPath); err == nil {
			// Exclude the specified path (e.g., ~/.claude/settings.local.json)
			cleanPath := filepath.Clean(settingsPath)
			if excludePath != "" && cleanPath == excludePath {
				continue
			}
			configs = append(configs, settingsPath)
		}
	}

	return configs
}

// DeduplicateConfig compares local settings against global settings
// and identifies duplicate entries.
func DeduplicateConfig(localPath string, global, local *claude.Settings) *DedupResult {
	result := &DedupResult{
		LocalPath: localPath,
	}

	// Find duplicates in each permission list
	result.DuplicateAllow = findDuplicates(local.Permissions.Allow, global.Permissions.Allow)
	result.DuplicateDeny = findDuplicates(local.Permissions.Deny, global.Permissions.Deny)
	result.DuplicateAsk = findDuplicates(local.Permissions.Ask, global.Permissions.Ask)

	// Check if local would become empty after removing duplicates
	uniqueSettings := local.Diff(global)
	result.SuggestDelete = uniqueSettings.IsEmpty()

	return result
}

// findDuplicates returns entries in local that also exist in global.
func findDuplicates(local, global []string) []string {
	if len(local) == 0 || len(global) == 0 {
		return nil
	}

	globalSet := make(map[string]struct{}, len(global))
	for _, v := range global {
		globalSet[v] = struct{}{}
	}

	var duplicates []string
	for _, v := range local {
		if _, exists := globalSet[v]; exists {
			duplicates = append(duplicates, v)
		}
	}

	return duplicates
}

// ApplyDedup applies the deduplication result to the local config file.
// If dryRun is true, returns without making changes.
func ApplyDedup(result *DedupResult, dryRun bool) error {
	if dryRun {
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(result.LocalPath); os.IsNotExist(err) {
		return nil
	}

	// If suggest delete, remove the file
	if result.SuggestDelete {
		return os.Remove(result.LocalPath)
	}

	// Otherwise, update the file by removing duplicates
	settings, err := claude.LoadSettings(result.LocalPath)
	if err != nil {
		return err
	}

	// Remove duplicates from each list
	settings.Permissions.Allow = removeEntries(settings.Permissions.Allow, result.DuplicateAllow)
	settings.Permissions.Deny = removeEntries(settings.Permissions.Deny, result.DuplicateDeny)
	settings.Permissions.Ask = removeEntries(settings.Permissions.Ask, result.DuplicateAsk)

	// Write updated settings back
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(result.LocalPath, data, 0600)
}

// removeEntries returns a new slice with specified entries removed.
func removeEntries(slice, toRemove []string) []string {
	if len(slice) == 0 {
		return nil
	}

	removeSet := make(map[string]struct{}, len(toRemove))
	for _, v := range toRemove {
		removeSet[v] = struct{}{}
	}

	var result []string
	for _, v := range slice {
		if _, exists := removeSet[v]; !exists {
			result = append(result, v)
		}
	}

	return result
}

// BuildDedupPreview creates a preview of configs to be deduplicated.
func BuildDedupPreview(results []DedupResult) *ui.Preview {
	preview := &ui.Preview{
		Title: "Config Deduplication",
	}

	for _, r := range results {
		var action ui.Action
		var description string

		if r.SuggestDelete {
			action = ui.ActionDelete
			description = "Empty after deduplication, will be deleted"
		} else {
			action = ui.ActionModify
			description = formatDuplicateDescription(r)
		}

		preview.Changes = append(preview.Changes, ui.Change{
			Action:      action,
			Path:        r.LocalPath,
			Description: description,
			Size:        0, // Config files are typically small
		})
	}

	return preview
}

// formatDuplicateDescription creates a description of duplicates found.
func formatDuplicateDescription(r DedupResult) string {
	total := r.TotalDuplicates()
	if total == 1 {
		return "1 duplicate entry to remove"
	}
	return fmt.Sprintf("%d duplicate entries to remove", total)
}

// BuildDedupPreviewVerbose creates a verbose preview of configs to be deduplicated.
func BuildDedupPreviewVerbose(results []DedupResult, globalPath string) *ui.Preview {
	preview := &ui.Preview{
		Title: "Config Deduplication",
	}

	for _, r := range results {
		var action ui.Action
		var description string

		if r.SuggestDelete {
			action = ui.ActionDelete
			description = formatVerboseDescription(r, globalPath, true)
		} else {
			action = ui.ActionModify
			description = formatVerboseDescription(r, globalPath, false)
		}

		preview.Changes = append(preview.Changes, ui.Change{
			Action:      action,
			Path:        r.LocalPath,
			Description: description,
			Size:        0,
		})
	}

	return preview
}

// formatVerboseDescription creates a verbose description listing all duplicates.
func formatVerboseDescription(r DedupResult, globalPath string, willDelete bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Duplicates of %s:\n", globalPath))

	if len(r.DuplicateAllow) > 0 {
		sb.WriteString("     allow: ")
		sb.WriteString(strings.Join(r.DuplicateAllow, ", "))
		sb.WriteString("\n")
	}

	if len(r.DuplicateDeny) > 0 {
		sb.WriteString("     deny: ")
		sb.WriteString(strings.Join(r.DuplicateDeny, ", "))
		sb.WriteString("\n")
	}

	if len(r.DuplicateAsk) > 0 {
		sb.WriteString("     ask: ")
		sb.WriteString(strings.Join(r.DuplicateAsk, ", "))
		sb.WriteString("\n")
	}

	if willDelete {
		sb.WriteString("     File will be deleted (no unique entries remain)")
	}

	return sb.String()
}
