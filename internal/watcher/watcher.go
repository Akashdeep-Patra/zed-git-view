// Package watcher monitors Git-internal state files for changes and notifies
// the TUI to refresh. Unlike naive approaches that recursively watch the
// entire working tree, this package watches only the ~5 files/directories
// inside .git that change on meaningful Git operations. This makes it safe
// for monorepos with 100k+ files where inotify/kqueue watches would be
// exhausted instantly.
//
// Watched paths:
//   - .git/index        → staging changes (git add/reset)
//   - .git/HEAD         → branch switches, commits
//   - .git/refs/heads   → local branch updates
//   - .git/refs/tags    → tag creation/deletion
//   - .git/refs/remotes → fetch/pull updates
//   - .git/MERGE_HEAD   → merge starts/ends
//   - .git/REBASE_HEAD  → rebase starts/ends
//   - .git/FETCH_HEAD   → fetch completions
//
// For working-tree changes (file edits), we rely on the user pressing 'r'
// (refresh) or on the debounced index-change event that git add/status
// triggers, which is the same strategy Lazygit uses.
package watcher

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event is sent when the watcher detects relevant Git state changes.
type Event struct{}

// Watch monitors critical Git-internal paths at repoRoot for state changes
// and sends Event values on the returned channel. Rapid bursts are coalesced
// via the debounce window.
//
// gitDir should be the absolute path to the .git directory (handles worktrees
// where .git is a file pointing elsewhere).
//
// Call the returned stop function to tear down the watcher.
func Watch(_, gitDir string, debounce time.Duration) (<-chan Event, func(), error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	// Core .git state files / directories to watch.
	targets := []string{
		gitDir,                              // catches HEAD, index, MERGE_HEAD etc.
		filepath.Join(gitDir, "refs"),       // catches all ref updates
		filepath.Join(gitDir, "refs/heads"), // local branch changes
		filepath.Join(gitDir, "refs/tags"),  // tag changes
	}

	// Also watch refs/remotes if it exists (fetch/pull updates).
	remotesDir := filepath.Join(gitDir, "refs/remotes")
	if info, err := os.Stat(remotesDir); err == nil && info.IsDir() {
		targets = append(targets, remotesDir)
		// Watch one level deep for per-remote dirs (e.g., refs/remotes/origin).
		entries, err := os.ReadDir(remotesDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					targets = append(targets, filepath.Join(remotesDir, e.Name()))
				}
			}
		}
	}

	// Watch packed-refs (large repos use this instead of individual ref files).
	packedRefs := filepath.Join(gitDir, "packed-refs")
	if _, err := os.Stat(packedRefs); err == nil {
		targets = append(targets, filepath.Dir(packedRefs))
	}

	for _, t := range targets {
		if info, statErr := os.Stat(t); statErr == nil && info.IsDir() {
			if addErr := w.Add(t); addErr != nil {
				// Non-fatal: some dirs may not exist yet.
				continue
			}
		} else if statErr == nil {
			// Watch parent dir for files.
			_ = w.Add(filepath.Dir(t))
		}
	}

	ch := make(chan Event, 1)
	done := make(chan struct{})

	// jitterRange adds randomness to the debounce to prevent the
	// "thundering herd" problem when multiple zgv instances watch
	// the same .git directory. Each instance fires at a slightly
	// different time, spreading the git subprocess load.
	jitterRange := debounce / 2 // 0 to 50% of debounce

	go func() {
		defer close(ch)
		var timer *time.Timer

		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if shouldIgnore(ev.Name) {
					continue
				}
				// Add random jitter to the debounce window.
				jitter := time.Duration(rand.Int64N(int64(jitterRange)))
				d := debounce + jitter
				if timer == nil {
					timer = time.NewTimer(d)
				} else {
					timer.Reset(d)
				}
			case <-timerChan(timer):
				timer = nil
				select {
				case ch <- Event{}:
				default:
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			case <-done:
				return
			}
		}
	}()

	stop := func() {
		close(done)
		_ = w.Close()
	}

	return ch, stop, nil
}

// timerChan returns the timer's channel, or a nil channel if timer is nil.
func timerChan(t *time.Timer) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

// shouldIgnore returns true for events that should not trigger a refresh.
func shouldIgnore(path string) bool {
	base := filepath.Base(path)

	// Git lock files: transient, mid-operation. NEVER trigger on these.
	// This is critical — git holds locks during status/add/commit and
	// we don't want to re-invoke git while it's holding a lock.
	if strings.HasSuffix(base, ".lock") {
		return true
	}

	// Editor swap/temp files that somehow end up in .git.
	if strings.HasSuffix(base, ".swp") || strings.HasSuffix(base, ".swo") ||
		strings.HasSuffix(base, "~") || strings.HasPrefix(base, ".#") {
		return true
	}

	// COMMIT_EDITMSG: this fires when you're typing a commit message
	// in an editor, not useful to refresh on.
	if base == "COMMIT_EDITMSG" {
		return true
	}

	// gc.log, fsmonitor, hooks are noise.
	if base == "gc.log" || strings.HasPrefix(base, "fsmonitor") {
		return true
	}

	return false
}
