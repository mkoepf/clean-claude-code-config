# cccc - Clean Claude Code Config

[![CI](https://github.com/mkoepf/clean-claude-code-config/actions/workflows/ci.yml/badge.svg)](https://github.com/mkoepf/clean-claude-code-config/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mkoepf/cccc)](https://goreportcard.com/report/github.com/mkoepf/cccc)
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/mkoepf/baae226f8088579c1405b06f7dd1a07a/raw/cccc-coverage.json)](https://github.com/mkoepf/clean-claude-code-config/actions/workflows/ci.yml)

A CLI utility to clean up Claude Code configuration by:

1. **Removing stale project session data** - when project directories no longer exist on disk
2. **Removing orphaned data** - empty sessions, orphan todos, file-history
3. **Deduplicating local config** - removes local settings that mirror global settings

## Features

- **Safe by default** - all destructive operations preview first and require explicit confirmation
- **Dry-run support** - see what would be cleaned without making changes
- **Audit logging** - all deletions are logged to `~/.claude/cccc-audit.log`

## Usage

```bash
cccc clean                          # Clean all (default: projects + orphans + config)
cccc clean projects [--dry-run]     # Remove stale project session data
cccc clean orphans [--dry-run]      # Remove orphaned data
cccc clean config [--dry-run]       # Deduplicate local configs against global settings
cccc list                           # List projects (default)
cccc list projects [--stale-only]   # List all projects with their status
cccc list orphans                   # List orphaned data without removing
cccc list config [--verbose]        # List duplicate config entries without removing
```

## Development & Testing

There is a Makefile to conveniently run various tests: 

```bash
# Using Make (recommended)
make test          # Run unit tests
make test-safety   # Run safety tests
make test-e2e      # Run E2E tests in Docker
make test-all      # Run all local tests
make quality       # Run full code quality checks
make help          # Show all available targets

# Or run directly
go test ./...                                    # Unit tests
go test -v -tags=safety ./test/safety/...        # Safety tests
docker build -t cccc-test -f test/Dockerfile . && docker run --rm cccc-test  # E2E tests
./scripts/code_quality.sh                        # Full code quality checks
```

## Terminology

- **Stale project**: A project directory registered in `~/.claude/projects/` whose corresponding source directory no longer exists on disk.
- **Orphaned data**: Files in `todos/`, `file-history/`, or `session-env/` that reference sessions which no longer exist, or empty session directories.

## Config Deduplication

Claude Code stores permissions in two places:
- **Global settings**: `~/.claude/settings.json` - applies to all projects
- **Local settings**: `<project>/.claude/settings.local.json` - project-specific overrides

Over time, local configs can accumulate entries that duplicate global settings. The `clean config` command removes these redundant entries.

### Example

**Global settings** (`~/.claude/settings.json`):
```json
{
  "permissions": {
    "allow": [
      "Bash(git:*)",
      "Bash(npm:*)"
    ],
    "deny": []
  }
}
```

**Local settings before** (`~/Code/myproject/.claude/settings.local.json`):
```json
{
  "permissions": {
    "allow": [
      "Bash(git:*)",
      "Bash(make:*)"
    ],
    "deny": []
  }
}
```

Running `cccc clean config` identifies `Bash(git:*)` as a duplicate (already in global) and removes it:

**Local settings after**:
```json
{
  "permissions": {
    "allow": [
      "Bash(make:*)"
    ],
    "deny": []
  }
}
```

If all entries in a local config are duplicates of global settings, the local file is deleted entirely.

## Claude Code Directory Layout

The tool was developed against Claude Code 2.0.62 and assumes the following
Claude Code directory structure:

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
