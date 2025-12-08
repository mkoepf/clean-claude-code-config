# CleanClaudeConfig (ccc)

A CLI utility to clean up Claude Code configuration by:

1. **Removing stale project session data** - when project directories no longer exist on disk
2. **Removing orphaned data** - empty sessions, orphan todos, file-history
3. **Deduplicating local config** - removes local settings that mirror global settings

## Features

- **Safe by default** - all destructive operations preview first and require explicit confirmation
- **Dry-run support** - see what would be cleaned without making changes
- **Audit logging** - all deletions are logged to `~/.claude/ccc-audit.log`

## Usage

```bash
ccc clean [--dry-run] [--yes]      # Clean all: stale projects, orphans, config duplicates
ccc clean projects [--dry-run]     # Remove stale project session data
ccc clean orphans [--dry-run]      # Remove orphaned data
ccc clean config [--dry-run]       # Deduplicate local configs against global settings
ccc list projects [--stale-only]   # List all projects with their status
ccc list orphans                   # List orphaned data without removing
```

## Implementation Status

### Phase 1: Core Library

| Component | Status | Description |
|-----------|--------|-------------|
| `internal/claude/sessions.go` | ✅ Complete | Parse session files, extract cwd |
| `internal/claude/paths.go` | ✅ Complete | Discover Claude directories |
| `internal/claude/projects.go` | ✅ Complete | Scan and analyze projects |
| `internal/claude/config.go` | ✅ Complete | Parse settings files |

### Phase 2: UI Components

| Component | Status | Description |
|-----------|--------|-------------|
| `internal/ui/preview.go` | ✅ Complete | Preview display formatting |
| `internal/ui/confirm.go` | ✅ Complete | Confirmation prompts |
| `internal/ui/audit.go` | ✅ Complete | Audit trail logging |

### Phase 3: Cleanup Operations

| Component | Status | Description |
|-----------|--------|-------------|
| `internal/cleaner/stale.go` | ✅ Complete | Find and clean stale projects |
| `internal/cleaner/orphans.go` | ✅ Complete | Find and clean orphans |
| `internal/cleaner/dedup.go` | ✅ Complete | Config deduplication |

### Phase 4: CLI Interface

| Component | Status | Description |
|-----------|--------|-------------|
| `cmd/ccc/main.go` | ✅ Complete | Full CLI implementation |

**Legend:** ✅ Complete | 🔲 Stub (tests written, not implemented) | ⬜ Not started

## Development

Tests are written before implementation (TDD).

```bash
# Run unit tests
go test ./...

# Run safety tests (verifies the tool never deletes existing projects)
go test -v -tags=safety ./test/safety/...

# Run full code quality checks (formatting, vet, tests, coverage)
./scripts/code_quality.sh
```

## Claude Code Directory Layout

The tool works with the standard Claude Code directory structure:

```
~/.claude/
├── settings.json          # Global settings
├── projects/              # Session data per project
│   └── {encoded-path}/    # e.g., -Users-mhk-Code-myproject
│       └── *.jsonl        # Session files (JSON Lines format)
├── todos/                 # Todo tracking files
├── file-history/          # File version history
└── session-env/           # Session environment
```
