# Contributing to zgv

Thank you for considering contributing to zgv! This document explains how to get
started and the standards we follow.

## Getting Started

### Prerequisites

- **Go 1.25+** ([install](https://go.dev/dl/))
- **Task** (optional) - `brew install go-task` or [taskfile.dev](https://taskfile.dev)
- **golangci-lint** (optional) - `brew install golangci-lint`

### Setup

```bash
git clone https://github.com/Akashdeep-Patra/zed-git-view.git
cd zed-git-view
go mod tidy
task build        # or: go build -o bin/zgv ./cmd/
```

### Running locally

```bash
# Quick run (no binary)
task dev

# Build and run
task run

# Run against a specific repo
task run -- --path /some/repo
```

### Running from Zed

Install global Zed integration once:

```bash
zgv zed install
```

Then in Zed command palette, run `task: spawn` and pick `zgv:*` tasks.

### Running from VS Code / Cursor

Install workspace integration:

```bash
zgv code install
```

This writes managed `zgv:*` tasks into `.vscode/tasks.json`, including an
auto-open-on-folder task for Git repositories.

## Development Workflow

### Branch naming

| Type | Format | Example |
|------|--------|---------|
| Feature | `feat/short-description` | `feat/file-tree-view` |
| Bug fix | `fix/short-description` | `fix/stash-parse-error` |
| Docs | `docs/short-description` | `docs/keybinding-table` |
| Refactor | `refactor/short-description` | `refactor/git-service` |

### Commit messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add file tree component to status view
fix: handle renamed files with spaces in path
docs: update keybinding reference in README
refactor: extract diff renderer to shared component
test: add parser tests for branch output
chore: bump bubbletea to v1.4.0
```

### Pull requests

1. Create a feature branch from `main`.
2. Make your changes with clear, atomic commits.
3. Ensure all checks pass: `task check` (format, lint, test).
4. Open a PR with a description of **what** and **why**.
5. One approval required before merge.

## Code Standards

### Architecture rules

- **Views never call `exec.Command` directly.** All git operations go through
  `git.Service` interface.
- **No import cycles.** Shared types live in `internal/common/`.
- **Keep `Update()` and `View()` fast.** Offload work to `tea.Cmd` functions.
- **Keep docs synchronized with runtime behavior.** If keybindings or commands
  change, update `README.md` and `CONTRIBUTING.md` in the same PR.

### Style

- Format with `gofumpt` (stricter than `gofmt`).
- Lint with `golangci-lint` using the project `.golangci.yml`.
- Use keyed struct literals (`{Key: "x", Desc: "y"}` not `{"x", "y"}`).
- Exported types/functions must have doc comments.

### Testing

```bash
task test           # full test suite with race detector
task test:short     # fast subset
task coverage       # HTML coverage report
task ci:local       # run CI jobs locally with act
```

- Unit test parsers and git output handlers.
- Integration tests can shell out to `git` in a temp repo.
- Use `testify/assert` for assertions.

## Adding a New View

1. Create `internal/ui/views/myview.go` implementing `common.View`.
2. Add a `TabID` constant in `internal/common/types.go`.
3. Add a `TabMeta` entry in `AllTabs`.
4. Wire it into `cmd/main.go` in the `viewMap`.
5. Add keybinding help entries via `ShortHelp()`.
6. Update `README.md` with the new view's shortcuts.

## Releasing

Releases are automated via GitHub Actions + GoReleaser:

1. Ensure your branch is clean and merged to `main`.
2. Run one command: `task release -- major|minor|patch`.
3. The release workflow builds binaries for all platforms, auto-generates release
   changelog notes from commits, publishes a GitHub Release with artifacts, and
   syncs those notes into `CHANGELOG.md` automatically.

If a previous release attempt partially uploaded assets for a tag, delete and
re-push the tag to force a clean run.

## Code of Conduct

Be respectful, constructive, and inclusive. We follow the
[Contributor Covenant](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).
