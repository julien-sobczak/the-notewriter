# The NoteWriter

The NoteWriter is a Go-based note management system using Markdown files with SQLite database indexing. It provides multiple CLI tools for managing notes, references, journals, and flashcards.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Dependencies and Setup

- Install ffmpeg: `sudo apt-get update && sudo apt-get install -y ffmpeg`
- Check Go version: `go version` (requires Go 1.23+)
- Download dependencies: `go mod download` -- takes 4-5 minutes. NEVER CANCEL. Set timeout to 10+ minutes.
- Verify modules: `go mod verify`

### Building

- Build all binaries: `make build` -- takes 1 minute. NEVER CANCEL. Set timeout to 3+ minutes.
- All builds require `--tags "fts5"` for SQLite FTS5 support
- Install additional tools: `make deps` -- installs `gocloc` for line counting

### Testing

- Run unit tests: `make test` -- takes less than 30 seconds. Set timeout to 2+ minutes.
- Run integration tests: `make test-all` -- takes less than 1 minute. Set timeout to 2+ minutes.

### Best Practices

- When writing tests that use `tr.WriteFile()`, use "‛" (double apostrophe U+201F) instead of "`" (backtick) for string delimiters to avoid concatenation issues. This method automatically rewrites backticks.
- Do not edit Markdown files under `examples/` directory with Copilot especially when explicitly told to do so.

### Documentation Website

- Install dependencies: `cd website && npm install` -- takes 1 minute. Set timeout to 5+ minutes.
- Build site: `npm run build` -- takes 13 seconds. Set timeout to 2+ minutes.
- Start dev server: `npm run dev` or `make docs` -- serves on http://localhost:4321/the-notewriter
- Uses Astro/Starlight framework

## Validation

### End-to-End Testing Scenarios

Always test complete workflows after making changes:

1. **Basic Commands Workflow:**

   ```bash
   cd examples/life/
   ../build/nt init
   ../build/nt status
   ../build/nt add . # takes ~2 minutes. NEVER CANCEL. Set timeout to 5+ minutes.
   ../build/nt status
   ../build/nt commit
   ../build/nt lint
   ```

2. **Additional Commands Testing:**

   ```bash
   ../build/ntreference --help
   ../build/ntjournal --help
   ../build/ntanki --help
   ```

3. **Documentation Testing:**

   ```bash
   cd website/
   npm run build
   npm run dev # Verify http://localhost:4321/the-notewriter loads
   ```

### Code Quality

- Format code: `go fmt ./...` -- some files typically need formatting

## Common Tasks

### Repository Root Structure
```
.
├── .github/           # GitHub workflows and templates
├── .gitignore
├── .ntignore         # NoteWriter ignore patterns
├── .vscode/          # VS Code settings
├── LICENSE
├── Makefile          # Main build targets
├── README.md         # Project overview
├── cmd/              # CLI tool implementations
│   ├── nt/           # Main note management tool
│   ├── nt-anki/      # Anki flashcard integration
│   ├── nt-reference/ # Reference generation from external sources
│   └── ntlite/       # Lightweight version (documentation-only)
├── e2e/              # End-to-end tests
├── examples/         # Sample note repositories for testing
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
├── internal/         # Internal packages
│   ├── core/         # Core note management logic
│   ├── helpers/      # Utility functions
│   ├── markdown/     # Markdown parsing
│   ├── medias/       # Media file handling
│   ├── reference/    # Reference management
│   └── remote/       # Remote repository sync
├── pkg/              # Public packages
└── website/          # Documentation site (Astro/Starlight)
```

### Main CLI Commands (`nt`)

```bash
nt init              # Initialize new note repository (.nt directory)
nt status            # Show repository status
nt add <files>       # Stage files for commit (can take 2+ minutes)
nt commit            # Commit staged changes
nt lint              # Validate note files format
nt diff              # Show changes between commits
nt reset             # Reset local database
nt pull/push         # Sync with remote repositories
nt gc                # Garbage collection
```


## Design

### Note Format

Notes are Markdown files with YAML Front Matter:

```markdown
---
title: "Note Title"
tags: ["tag1", "tag2"]
source: "https://example.com"
---

# Note Title

## Note: Section Title

Content goes here.

## Quote: Famous Quote

> This is a quote
```

Supported note types are `Note`, `Quote`, `TODO`, `Flashcard`, `Artwork` but custom note types are supported.

### Key Files and Configurations

- `.nt/config.jsonnet` - Main configuration file
- `.nt/nt.libsonnet` - Library functions
- `.nt/database.db` - SQLite database with FTS5 indexing
- `.ntignore` - Files to ignore during processing
- Makefile targets: `build`, `test`, `test-all`, `docs`, `deps`, `cloc`

## Critical Build and Test Timing

- NEVER CANCEL any build or test command before specified timeouts

- `go mod download`: 4-5 minutes (timeout: 10+ minutes)
- `make build`: 1 minute (timeout: 3+ minutes)
- `make test`: 12-15 seconds (timeout: 2+ minutes)
- `nt add .`: 2+ minutes for example repo (timeout: 5+ minutes)
- `npm install`: 1 minute (timeout: 5+ minutes)

## Dependencies

- Always use `--tags "fts5"` when building Go binaries
- The `examples/` directory provides a good test environment
- `ffmpeg` dependency required for media file processing
- SQLite with FTS5 extension required for full-text search
- Node.js 20+ required for documentation website
