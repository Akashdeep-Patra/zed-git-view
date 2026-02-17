# zgv — A Modern TUI Git Client

[![CI](https://github.com/Akashdeep-Patra/zed-git-view/actions/workflows/ci.yml/badge.svg)](https://github.com/Akashdeep-Patra/zed-git-view/actions/workflows/ci.yml)
[![Release](https://github.com/Akashdeep-Patra/zed-git-view/actions/workflows/release.yml/badge.svg)](https://github.com/Akashdeep-Patra/zed-git-view/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/Akashdeep-Patra/zed-git-view)](https://goreportcard.com/report/github.com/Akashdeep-Patra/zed-git-view)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A keyboard-first, terminal-based Git client built with Go and the [Charm](https://charm.sh) ecosystem. Designed to run inside [Zed's](https://zed.dev) integrated terminal (or any terminal emulator).

## Features

| View | Key | Description |
|------|-----|-------------|
| **Status** | `1` | Stage/unstage files, commit, discard changes, diff preview |
| **Log** | `2` | Commit graph with ASCII art, commit detail panel |
| **Diff** | `3` | Inline and side-by-side diff viewer with syntax colouring |
| **Branches** | `4` | Create, switch, rename, delete, merge branches |
| **Stash** | `5` | Save, pop, apply, drop stashes with diff preview |
| **Remotes** | `6` | Fetch, pull, push with remote selection |
| **Rebase** | `7` | Start interactive rebase, continue, abort |
| **Conflicts** | `8` | View conflict files, mark resolved, show diff |
| **Worktrees** | `9` | Add and remove linked working trees |
| **Bisect** | `0` | Interactive binary search for bug-introducing commits |

## Installation

### Homebrew (macOS / Linux)

```bash
brew install Akashdeep-Patra/tap/zgv
```

### Go install

```bash
# Requires Go 1.22+
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
| `1`-`9`, `0` | Switch to tab |
| `tab` / `shift+tab` | Next / previous tab |
| `j` / `k` | Navigate down / up |
| `g` / `G` | Go to top / bottom |
| `?` | Toggle help overlay |
| `r` | Refresh data |
| `q` / `ctrl+c` | Quit |

### Status View

| Key | Action |
|-----|--------|
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
# Update CHANGELOG.md, commit, then tag:
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0

# Or use the Taskfile shortcut:
task tag -- v0.2.0
```

Pushing a `v*` tag triggers the release workflow which:
1. Runs all CI checks.
2. Builds binaries for 6 platform/arch combos.
3. Creates `.deb`, `.rpm`, `.apk` packages.
4. Generates SHA-256 checksums.
5. Signs artifacts with cosign (keyless, Sigstore).
6. Generates an SBOM (Software Bill of Materials).
7. Publishes a GitHub Release with all artifacts.
8. Updates the Homebrew formula.

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
| **CI** | push/PR to `main` | lint, test (3 OS), vet, build (6 platforms) |
| **Release** | `v*` tag push | goreleaser, sign, SBOM, homebrew, packages |

GitHub Actions handles everything automatically. PRs cannot merge unless all
CI checks pass (configure branch protection rules for enforcement).

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
  ci.yml                 CI: lint, test, vet, build on every push/PR
  release.yml            Release: goreleaser on tag push
.goreleaser.yml          Multi-platform build + package + publish config
Taskfile.yml             Development task runner (build, test, lint, tag, etc.)
.golangci.yml            Linter configuration
CHANGELOG.md             Keep a Changelog format
CONTRIBUTING.md          Contribution guidelines
LICENSE                  MIT
```

## Development

```bash
# Install task runner
brew install go-task

# Common commands
task build          # Build binary (with version metadata)
task dev            # Run with go run
task test           # Run tests with race detector
task lint           # Run golangci-lint
task check          # fmt + vet + lint + test
task snapshot       # GoReleaser snapshot (local, no publish)
task release:dry    # Full release dry-run
task version        # Print current version from git tags
task tag -- v0.2.0  # Tag and push a release
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, workflow, and coding standards.

## License

[MIT](LICENSE)
