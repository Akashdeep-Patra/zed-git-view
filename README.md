# zgv — A Modern TUI Git Client

[![CI](https://github.com/Akashdeep-Patra/zed-git-view/actions/workflows/ci.yml/badge.svg)](https://github.com/Akashdeep-Patra/zed-git-view/actions/workflows/ci.yml)
[![Release](https://github.com/Akashdeep-Patra/zed-git-view/actions/workflows/release.yml/badge.svg)](https://github.com/Akashdeep-Patra/zed-git-view/releases)
[![GitHub Release](https://img.shields.io/github/v/release/Akashdeep-Patra/zed-git-view?label=latest)](https://github.com/Akashdeep-Patra/zed-git-view/releases/latest)
[![Homebrew](https://img.shields.io/badge/homebrew-Akashdeep--Patra%2Ftap-orange)](https://github.com/Akashdeep-Patra/homebrew-tap)
[![Go Report Card](https://goreportcard.com/badge/github.com/Akashdeep-Patra/zed-git-view)](https://goreportcard.com/report/github.com/Akashdeep-Patra/zed-git-view)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A keyboard-first, terminal-based Git client built with Go and the [Charm](https://charm.sh) ecosystem. Designed to run inside [Zed's](https://zed.dev) integrated terminal (or any terminal emulator).

## Features

| View | Direct Shortcut | Description |
|------|-----------------|-------------|
| **Status** | `alt+s` | Stage/unstage files, commit, discard changes, diff preview |
| **Log** | `alt+l` | Commit graph with ASCII art, commit detail panel |
| **Diff** | `alt+d` | Inline and side-by-side diff viewer with syntax colouring |
| **Branches** | `alt+b` | Create, switch, rename, delete, merge branches |
| **Stash** | `alt+t` | Save, pop, apply, drop stashes with diff preview |
| **Remotes** | `alt+m` | Fetch, pull, push with remote selection |
| **Rebase** | `alt+e` | Start interactive rebase, continue, abort |
| **Conflicts** | `alt+x` | View conflict files, mark resolved, show diff |
| **Worktrees** | `alt+w` | Add and remove linked working trees |
| **Bisect** | `alt+i` | Interactive binary search for bug-introducing commits |

## Installation

### Homebrew (macOS / Linux)

```bash
brew install Akashdeep-Patra/tap/zgv
```

### Go install

```bash
# Requires Go 1.25+
go install github.com/Akashdeep-Patra/zed-git-view/cmd@latest
```

### Download binary

Pre-built binaries for Linux, macOS, Windows (amd64 + arm64) are available on
the [Releases](https://github.com/Akashdeep-Patra/zed-git-view/releases) page.

Each release includes SHA-256 checksums and SBOM for verification:

```bash
# Verify checksum after download
sha256sum -c checksums.txt --ignore-missing
```

### Linux packages

`.deb`, `.rpm`, and `.apk` packages are attached to each GitHub Release.

```bash
# Debian / Ubuntu
sudo dpkg -i zgv_*.deb

# Fedora / RHEL
sudo rpm -i zgv_*.rpm

# Alpine
sudo apk add --allow-untrusted zgv_*.apk
```

### Build from source

```bash
git clone https://github.com/Akashdeep-Patra/zed-git-view.git
cd zed-git-view
task build          # or: go build -o bin/zgv ./cmd/
./bin/zgv
```

## Usage

```bash
# Run in the current directory
zgv

# Run in a specific repo
zgv --path /path/to/repo

# Print version
zgv version
zgv version --json

# Shell completions
zgv completion bash   # also: zsh, fish, powershell
```

## Keyboard Shortcuts

### Global

| Key | Action |
|-----|--------|
| `left` / `right` | Previous / next tab |
| `h` / `l` | Previous / next tab (vim-style alias) |
| `alt+s` / `alt+d` / `alt+l` / `alt+b` / `alt+m` / `alt+t` / `alt+e` / `alt+x` / `alt+w` / `alt+i` | Jump to specific tab |
| `up` / `down` | Navigate up / down |
| `home` / `end` | Go to top / bottom |
| `pgup` / `pgdn` (`ctrl+u` / `ctrl+d`) | Page up / down |
| `enter` | Confirm action |
| `esc` | Back / close overlay |
| `?` | Toggle help overlay |
| `r` | Refresh data |
| `q` / `ctrl+c` | Quit |

### Status View

| Key | Action |
|-----|--------|
| `tab` | Switch file list / diff pane focus |
| `s` / `S` | Stage file / stage all |
| `u` / `U` | Unstage file / unstage all |
| `x` | Discard changes |
| `c` | Commit (ctrl+s to confirm) |
| `d` / `enter` | Preview diff |

### Diff View

| Key | Action |
|-----|--------|
| `v` | Toggle inline / side-by-side |
| `ctrl+d` / `ctrl+u` | Page down / up |

### Branch View

| Key | Action |
|-----|--------|
| `enter` | Switch to branch |
| `n` | Create new branch |
| `R` | Rename branch |
| `D` | Delete branch |
| `m` | Merge into current |

## Zed IDE Integration

This project includes workspace-native Zed tasks in `.zed/tasks.json`.

### Run zgv from Zed automatically

1. Open this repository in Zed.
2. Open command palette and run `task: spawn`.
3. Run one of:
   - `zgv: open (current worktree)`
   - `zgv: dev hot reload`
   - `zgv: check (fmt+vet+lint+test)`

You can also bind a shortcut in your Zed keymap to run tasks directly:

```json
{
  "context": "Workspace",
  "bindings": {
    "alt-g": ["task::Spawn", { "task_name": "zgv: open (current worktree)" }]
  }
}
```

### Notes

- `task dev` uses `watchexec` to restart `go run ./cmd` when `.go` files change.
- During runtime, `zgv` auto-refreshes from `.git` state changes via `fsnotify`.
- For best behavior in monorepos, use `zgv: open (current worktree)` so the repo
  path is always explicit (`$ZED_WORKTREE_ROOT`).

## Configuration

Configuration file: `~/.config/zgv/config.yaml`

```yaml
theme: dark
max_log_entries: 200
confirm_destructive: true
diff_context_lines: 3
side_by_side_diff: false
```

Environment variables (prefixed with `ZGV_`):

```bash
export ZGV_MAX_LOG_ENTRIES=500
```

## Versioning

This project follows [Semantic Versioning](https://semver.org/).

- Version is injected at build time via Go ldflags.
- `zgv version` prints version, commit hash, build date, Go version, and OS/arch.
- `zgv version --json` outputs machine-readable version info.
- Tags follow the format `v{major}.{minor}.{patch}` (e.g., `v0.1.0`).

### Creating a release

```bash
# One command: runs checks, increments semver, tags, pushes
task release -- patch   # v0.1.0 -> v0.1.1
task release -- minor   # v0.1.0 -> v0.2.0
task release -- major   # v0.1.0 -> v1.0.0
```

What this does:
1. Runs `task check` (fmt, vet, lint, test).
2. Reads latest `v*` tag and computes the next semver based on `major|minor|patch`.
3. Pushes the tag to origin.

Pushing a `v*` tag then triggers the release workflow which:
1. Builds multi-platform binaries and archives.
2. Creates `.deb`, `.rpm`, `.apk` packages.
3. Generates SHA-256 checksums.
4. Signs artifacts with cosign (keyless, Sigstore).
5. Generates SBOM artifacts.
6. Auto-generates release changelog notes from commits.
7. Publishes a GitHub Release with all artifacts.
8. Updates the Homebrew formula and tap README.
9. Syncs detailed release notes into `CHANGELOG.md`.

If a release run is retried or a tag is re-pushed, GitHub can report
`422 already_exists` during asset upload for files already present on the
release. In that case, the safest recovery is:

```bash
git push origin :refs/tags/vX.Y.Z
git push origin vX.Y.Z
```

## Tech Stack

- **Go** — fast, single binary, no runtime deps
- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** — Elm Architecture TUI framework
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — CSS-like terminal styling
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — pre-built TUI components
- **[Cobra](https://cobra.dev)** — CLI framework with completions
- **[Viper](https://github.com/spf13/viper)** — configuration management
- **Git CLI** — shells out to `git` for maximum compatibility

## CI/CD

| Workflow | Trigger | Jobs |
|----------|---------|------|
| **CI** | `v*` tag push | lint, test (3 OS), vet, build (6 platforms) |
| **Release** | `v*` tag push | goreleaser, sign, SBOM, homebrew, packages |

GitHub Actions handles release-tag automation end to end.

## Project Structure

```
cmd/main.go              Entry point (Cobra CLI, version, completions)
internal/
  app/                   App model, keybindings, orchestration
  common/                Shared types (TabID, messages, View interface)
  config/                Viper-based configuration
  git/                   Git service interface + CLI implementation
  ui/
    theme.go             Catppuccin-inspired dark theme
    layout.go            Layout helpers
    components/          Shared components (tabs, statusbar, help, dialog, side-by-side diff)
    views/               One file per tab (status, log, diff, branches, stash, remotes, rebase, conflicts, worktrees, bisect)
.github/workflows/
  ci.yml                 CI: lint, test, vet, build on release tags
  release.yml            Release: goreleaser on tag push
.goreleaser.yml          Multi-platform build + package + publish config
Taskfile.yml             Development task runner (build, test, lint, tag, etc.)
.golangci.yml            Linter configuration
CHANGELOG.md             Auto-synced release history from GitHub release notes
CONTRIBUTING.md          Contribution guidelines
LICENSE                  MIT
```

## Development

```bash
# Install task runner
brew install go-task

# Common commands
task build          # Build binary (with version metadata)
task dev            # Hot reload via watchexec
task test           # Run tests with race detector
task lint           # Run golangci-lint
task check          # fmt + vet + lint + test
task snapshot       # GoReleaser snapshot (local, no publish)
task release:dry    # Full release dry-run
task ci:local       # Run CI jobs locally via act
task version        # Print current version from git tags
task release -- patch   # Check + auto-bump + tag + push
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, workflow, and coding standards.

## License

[MIT](LICENSE)
