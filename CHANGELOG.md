# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial project scaffold with Taskfile, golangci-lint, GoReleaser
- Git service interface with full CLI implementation (40+ operations)
- Catppuccin Mocha-inspired dark theme with 60+ Lipgloss styles
- 10 interactive views:
  - **Status**: stage/unstage files, commit with textarea, discard, diff preview
  - **Log**: ASCII commit graph, commit detail panel, ref decorations
  - **Diff**: inline and side-by-side modes with syntax colouring
  - **Branches**: create, switch, rename, delete, merge
  - **Stash**: save, pop, apply, drop with diff preview
  - **Remotes**: fetch, fetch all, pull, push
  - **Rebase**: start interactive rebase, continue, abort
  - **Conflicts**: list conflicts, mark resolved, show diff
  - **Worktrees**: add and remove linked working trees
  - **Bisect**: start, mark good/bad, reset, log viewer
- Tab navigation with `1-9, 0` hotkeys and `tab/shift+tab`
- Vim-style navigation (`j/k/g/G`)
- Full help overlay (`?`) with contextual keybindings per view
- Status bar with branch, ahead/behind, merge/rebase state
- Modal dialog system (confirm + text input)
- Cobra CLI with `--path` flag and `version` subcommand
- Viper configuration with `~/.config/zgv/config.yaml` support
- GitHub Actions CI (lint, test, vet, build) and release workflows
- GoReleaser multi-platform builds (linux/darwin/windows x amd64/arm64)
- Shell completions for bash, zsh, fish, powershell

## [0.1.0] - 2026-02-17

### Added

- First release with all core features.

[Unreleased]: https://github.com/Akashdeep-Patra/zed-git-view/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Akashdeep-Patra/zed-git-view/releases/tag/v0.1.0
