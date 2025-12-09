package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mhk/ccc/internal/claude"
	"github.com/mhk/ccc/internal/cleaner"
	"github.com/mhk/ccc/internal/ui"
)

// Args represents parsed command-line arguments.
type Args struct {
	Command    string // "clean", "list", ""
	Subcommand string // "projects", "orphans", "config", ""
	DryRun     bool
	Yes        bool
	StaleOnly  bool
	Verbose    bool
	Help       bool
}

func main() {
	code := runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

// runCLI is the main entry point for the CLI, testable via io.Reader/Writer.
func runCLI(osArgs []string, stdin io.Reader, stdout, stderr io.Writer) int {
	args, err := parseArgs(osArgs)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}

	if args.Help {
		printHelp(stdout)
		return 0
	}

	// Discover Claude paths
	paths, err := claude.DiscoverPaths("")
	if err != nil {
		fmt.Fprintln(stderr, "Error discovering Claude paths:", err)
		return 1
	}

	switch args.Command {
	case "clean":
		return handleClean(args, paths, stdin, stdout, stderr)
	case "list":
		return handleList(args, paths, stdout, stderr)
	default:
		printHelp(stdout)
		return 0
	}
}

// parseArgs parses command-line arguments into Args struct.
func parseArgs(osArgs []string) (*Args, error) {
	args := &Args{}

	if len(osArgs) == 0 {
		args.Help = true
		return args, nil
	}

	i := 0
	for i < len(osArgs) {
		arg := osArgs[i]

		switch arg {
		case "-h", "--help", "help":
			args.Help = true
			return args, nil
		case "--dry-run":
			args.DryRun = true
		case "-y", "--yes":
			args.Yes = true
		case "--stale-only":
			args.StaleOnly = true
		case "-v", "--verbose":
			args.Verbose = true
		case "clean", "list":
			if args.Command == "" {
				args.Command = arg
			} else {
				args.Subcommand = arg
			}
		case "projects", "orphans", "config":
			args.Subcommand = arg
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			return nil, fmt.Errorf("unknown command: %s", arg)
		}
		i++
	}

	return args, nil
}

// printHelp prints the usage information.
func printHelp(w io.Writer) {
	fmt.Fprintln(w, "ccc - CleanClaudeConfig")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "A CLI utility to clean up Claude Code configuration.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ccc clean [--dry-run] [--yes]      Clean all: stale projects, orphans, config duplicates")
	fmt.Fprintln(w, "  ccc clean projects [--dry-run]     Remove stale project session data")
	fmt.Fprintln(w, "  ccc clean orphans [--dry-run]      Remove orphaned data")
	fmt.Fprintln(w, "  ccc clean config [--dry-run]       Deduplicate local configs against global settings")
	fmt.Fprintln(w, "  ccc list projects [--stale-only]   List all projects with their status")
	fmt.Fprintln(w, "  ccc list orphans                   List orphaned data without removing")
	fmt.Fprintln(w, "  ccc list config [--verbose]        List duplicate config entries without removing")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --dry-run      Show what would be cleaned without making changes")
	fmt.Fprintln(w, "  --yes, -y      Skip confirmation prompts")
	fmt.Fprintln(w, "  --verbose, -v  Show detailed output (e.g., list duplicate entries)")
	fmt.Fprintln(w, "  --stale-only   Show only stale projects (with list command)")
	fmt.Fprintln(w, "  --help, -h     Show this help message")
}

// handleClean handles the "clean" command and subcommands.
func handleClean(args *Args, paths *claude.Paths, stdin io.Reader, stdout, stderr io.Writer) int {
	switch args.Subcommand {
	case "projects":
		return cleanProjects(args, paths, stdin, stdout, stderr)
	case "orphans":
		return cleanOrphans(args, paths, stdin, stdout, stderr)
	case "config":
		return cleanConfig(args, paths, stdin, stdout, stderr)
	case "":
		// Clean all
		code := cleanProjects(args, paths, stdin, stdout, stderr)
		if code != 0 {
			return code
		}
		code = cleanOrphans(args, paths, stdin, stdout, stderr)
		if code != 0 {
			return code
		}
		return cleanConfig(args, paths, stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown clean subcommand: %s\n", args.Subcommand)
		return 1
	}
}

// handleList handles the "list" command and subcommands.
func handleList(args *Args, paths *claude.Paths, stdout, stderr io.Writer) int {
	switch args.Subcommand {
	case "projects", "":
		return listProjects(args, paths, stdout, stderr)
	case "orphans":
		return listOrphans(paths, stdout, stderr)
	case "config":
		return listConfig(args, paths, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown list subcommand: %s\n", args.Subcommand)
		return 1
	}
}

// cleanProjects finds and removes stale project session data.
func cleanProjects(args *Args, paths *claude.Paths, stdin io.Reader, stdout, stderr io.Writer) int {
	projects, err := claude.ScanProjects(paths.Projects)
	if err != nil {
		fmt.Fprintln(stderr, "Error scanning projects:", err)
		return 1
	}

	stale := cleaner.FindStaleProjects(projects)
	if len(stale) == 0 {
		fmt.Fprintln(stdout, "No stale projects found.")
		return 0
	}

	// Build kept list (non-stale)
	var kept []claude.Project
	staleSet := make(map[string]bool)
	for _, p := range stale {
		staleSet[p.EncodedName] = true
	}
	for _, p := range projects {
		if !staleSet[p.EncodedName] {
			kept = append(kept, p)
		}
	}

	preview := cleaner.BuildStalePreview(stale, kept)

	if args.DryRun {
		fmt.Fprintln(stdout, "[DRY RUN]")
		_ = preview.Display(stdout)
		return 0
	}

	confirmed, err := ui.ConfirmChanges(preview, stdin, stdout, args.Yes)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !confirmed {
		return 0
	}

	// Create audit logger
	auditLogger, err := ui.NewAuditLogger(ui.DefaultAuditLogPath(paths.Root))
	if err != nil {
		fmt.Fprintln(stderr, "Warning: could not create audit log:", err)
	} else {
		defer auditLogger.Close()
	}

	// Perform cleanup
	var totalSaved int64
	for _, p := range stale {
		result, err := cleaner.CleanStaleProject(paths.Projects, p, false)
		if err != nil {
			fmt.Fprintf(stderr, "Error cleaning project %s: %v\n", p.ActualPath, err)
			continue
		}
		totalSaved += result.SizeSaved

		if auditLogger != nil {
			_ = auditLogger.Log(ui.ActionDelete, p.ActualPath, result.SizeSaved)
		}
	}

	fmt.Fprintf(stdout, "Cleaned %d stale projects, freed %s\n", len(stale), ui.FormatSize(totalSaved))
	return 0
}

// cleanOrphans finds and removes orphaned data.
func cleanOrphans(args *Args, paths *claude.Paths, stdin io.Reader, stdout, stderr io.Writer) int {
	// Get valid session IDs from projects
	projects, err := claude.ScanProjects(paths.Projects)
	if err != nil {
		fmt.Fprintln(stderr, "Error scanning projects:", err)
		return 1
	}

	var validSessionIDs []string
	for _, p := range projects {
		validSessionIDs = append(validSessionIDs, p.SessionIDs...)
	}

	orphans, err := cleaner.FindOrphans(paths, validSessionIDs)
	if err != nil {
		fmt.Fprintln(stderr, "Error finding orphans:", err)
		return 1
	}

	if len(orphans) == 0 {
		fmt.Fprintln(stdout, "No orphaned data found.")
		return 0
	}

	preview := cleaner.BuildOrphanPreview(orphans)

	if args.DryRun {
		fmt.Fprintln(stdout, "[DRY RUN]")
		_ = preview.Display(stdout)
		return 0
	}

	confirmed, err := ui.ConfirmChanges(preview, stdin, stdout, args.Yes)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !confirmed {
		return 0
	}

	// Create audit logger
	auditLogger, err := ui.NewAuditLogger(ui.DefaultAuditLogPath(paths.Root))
	if err != nil {
		fmt.Fprintln(stderr, "Warning: could not create audit log:", err)
	} else {
		defer auditLogger.Close()
	}

	// Perform cleanup
	results, err := cleaner.CleanOrphans(orphans, false)
	if err != nil {
		fmt.Fprintln(stderr, "Error cleaning orphans:", err)
		return 1
	}

	var totalSaved int64
	for _, r := range results {
		totalSaved += r.SizeSaved
		if auditLogger != nil {
			_ = auditLogger.Log(ui.ActionDelete, r.Path, r.SizeSaved)
		}
	}

	fmt.Fprintf(stdout, "Cleaned %d orphaned items, freed %s\n", len(results), ui.FormatSize(totalSaved))
	return 0
}

// cleanConfig deduplicates local configs against global settings.
func cleanConfig(args *Args, paths *claude.Paths, stdin io.Reader, stdout, stderr io.Writer) int {
	// Load global settings
	global, err := claude.LoadSettings(paths.Settings)
	if err != nil {
		fmt.Fprintln(stderr, "Error loading global settings:", err)
		return 1
	}

	// Get project paths from scanned projects for fast config lookup
	projects, err := claude.ScanProjects(paths.Projects)
	if err != nil {
		fmt.Fprintln(stderr, "Error scanning projects:", err)
		return 1
	}

	// Extract unique project paths
	var projectPaths []string
	for _, p := range projects {
		if p.ActualPath != "" {
			projectPaths = append(projectPaths, p.ActualPath)
		}
	}

	// Find local configs only in known project directories (fast)
	// Exclude ~/.claude/settings.local.json (if home dir is a project, it shouldn't be treated as a local config)
	homeLocalSettings := filepath.Join(paths.Root, "settings.local.json")
	localConfigs := cleaner.FindLocalConfigsFromProjects(projectPaths, homeLocalSettings)

	if len(localConfigs) == 0 {
		fmt.Fprintln(stdout, "No local configs found.")
		return 0
	}

	// Analyze each local config
	var results []cleaner.DedupResult
	for _, configPath := range localConfigs {
		local, err := claude.LoadSettings(configPath)
		if err != nil {
			fmt.Fprintf(stderr, "Warning: could not load %s: %v\n", configPath, err)
			continue
		}

		result := cleaner.DeduplicateConfig(configPath, global, local)
		if result.HasDuplicates() || result.SuggestDelete {
			results = append(results, *result)
		}
	}

	if len(results) == 0 {
		fmt.Fprintln(stdout, "No duplicate configs found.")
		return 0
	}

	// Use verbose preview if requested
	var preview *ui.Preview
	if args.Verbose {
		preview = cleaner.BuildDedupPreviewVerbose(results, paths.Settings)
	} else {
		preview = cleaner.BuildDedupPreview(results)
	}

	if args.DryRun {
		fmt.Fprintln(stdout, "[DRY RUN]")
		_ = preview.Display(stdout)
		return 0
	}

	confirmed, err := ui.ConfirmChanges(preview, stdin, stdout, args.Yes)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	if !confirmed {
		return 0
	}

	// Create audit logger
	auditLogger, err := ui.NewAuditLogger(ui.DefaultAuditLogPath(paths.Root))
	if err != nil {
		fmt.Fprintln(stderr, "Warning: could not create audit log:", err)
	} else {
		defer auditLogger.Close()
	}

	// Apply deduplication
	for _, r := range results {
		if err := cleaner.ApplyDedup(&r, false); err != nil {
			fmt.Fprintf(stderr, "Error deduplicating %s: %v\n", r.LocalPath, err)
			continue
		}
		if auditLogger != nil {
			if r.SuggestDelete {
				_ = auditLogger.Log(ui.ActionDelete, r.LocalPath, 0)
			} else {
				_ = auditLogger.Log(ui.ActionModify, r.LocalPath, 0)
			}
		}
	}

	fmt.Fprintf(stdout, "Deduplicated %d config files\n", len(results))
	return 0
}

// listProjects lists all projects and their status.
func listProjects(args *Args, paths *claude.Paths, stdout, stderr io.Writer) int {
	projects, err := claude.ScanProjects(paths.Projects)
	if err != nil {
		fmt.Fprintln(stderr, "Error scanning projects:", err)
		return 1
	}

	if len(projects) == 0 {
		fmt.Fprintln(stdout, "No projects found.")
		return 0
	}

	stale := cleaner.FindStaleProjects(projects)
	staleSet := make(map[string]bool)
	for _, p := range stale {
		staleSet[p.EncodedName] = true
	}

	fmt.Fprintln(stdout, "Projects:")
	for _, p := range projects {
		isStale := staleSet[p.EncodedName]

		// Skip non-stale if --stale-only
		if args.StaleOnly && !isStale {
			continue
		}

		status := "OK"
		if isStale {
			status = "STALE"
		}

		path := p.ActualPath
		if path == "" {
			path = "(unknown path)"
		}

		fmt.Fprintf(stdout, "  [%s] %s\n", status, path)
		fmt.Fprintf(stdout, "        %d files, %s, last used: %s\n",
			p.FileCount, ui.FormatSize(p.TotalSize), p.LastUsed.Format("2006-01-02"))
	}

	fmt.Fprintf(stdout, "\nTotal: %d projects (%d stale)\n", len(projects), len(stale))
	return 0
}

// listOrphans lists orphaned data without removing it.
func listOrphans(paths *claude.Paths, stdout, stderr io.Writer) int {
	// Get valid session IDs from projects
	projects, err := claude.ScanProjects(paths.Projects)
	if err != nil {
		fmt.Fprintln(stderr, "Error scanning projects:", err)
		return 1
	}

	var validSessionIDs []string
	for _, p := range projects {
		validSessionIDs = append(validSessionIDs, p.SessionIDs...)
	}

	orphans, err := cleaner.FindOrphans(paths, validSessionIDs)
	if err != nil {
		fmt.Fprintln(stderr, "Error finding orphans:", err)
		return 1
	}

	if len(orphans) == 0 {
		fmt.Fprintln(stdout, "No orphaned data found.")
		return 0
	}

	preview := cleaner.BuildOrphanPreview(orphans)
	_ = preview.Display(stdout)

	return 0
}

// listConfig lists duplicate config entries without removing them.
func listConfig(args *Args, paths *claude.Paths, stdout, stderr io.Writer) int {
	// Load global settings
	global, err := claude.LoadSettings(paths.Settings)
	if err != nil {
		fmt.Fprintln(stderr, "Error loading global settings:", err)
		return 1
	}

	// Get project paths from scanned projects for fast config lookup
	projects, err := claude.ScanProjects(paths.Projects)
	if err != nil {
		fmt.Fprintln(stderr, "Error scanning projects:", err)
		return 1
	}

	// Extract unique project paths
	var projectPaths []string
	for _, p := range projects {
		if p.ActualPath != "" {
			projectPaths = append(projectPaths, p.ActualPath)
		}
	}

	// Find local configs only in known project directories (fast)
	homeLocalSettings := filepath.Join(paths.Root, "settings.local.json")
	localConfigs := cleaner.FindLocalConfigsFromProjects(projectPaths, homeLocalSettings)

	if len(localConfigs) == 0 {
		fmt.Fprintln(stdout, "No local configs found.")
		return 0
	}

	// Analyze each local config
	var results []cleaner.DedupResult
	for _, configPath := range localConfigs {
		local, err := claude.LoadSettings(configPath)
		if err != nil {
			fmt.Fprintf(stderr, "Warning: could not load %s: %v\n", configPath, err)
			continue
		}

		result := cleaner.DeduplicateConfig(configPath, global, local)
		if result.HasDuplicates() || result.SuggestDelete {
			results = append(results, *result)
		}
	}

	if len(results) == 0 {
		fmt.Fprintln(stdout, "No duplicate configs found.")
		return 0
	}

	// Use verbose preview if requested
	var preview *ui.Preview
	if args.Verbose {
		preview = cleaner.BuildDedupPreviewVerbose(results, paths.Settings)
	} else {
		preview = cleaner.BuildDedupPreview(results)
	}

	_ = preview.Display(stdout)

	return 0
}
