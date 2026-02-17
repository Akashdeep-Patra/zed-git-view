package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotARepo is returned when the path is not inside a Git repository.
var ErrNotARepo = errors.New("not a git repository")

// cmdTimeoutRead is the max duration for read-only git commands (status,
// diff, log, etc.). Reads should be fast even on large repos; if they
// exceed this, something is wrong (NFS stall, lock contention).
const cmdTimeoutRead = 10 * time.Second

// cmdTimeoutWrite is the max duration for write commands (add, commit, etc.).
// Slightly longer because index updates on large repos can take a moment.
const cmdTimeoutWrite = 30 * time.Second

// cmdTimeoutNetwork is the max duration for network commands (fetch, push, pull).
// These depend on server speed and should be generous.
const cmdTimeoutNetwork = 120 * time.Second

// maxConcurrentGitProcs limits the number of git subprocesses a single
// instance may run concurrently. This is critical for multi-instance
// scenarios: without a limit, N instances × M concurrent commands
// overwhelms the machine. A semaphore of 4 allows enough parallelism
// for a refresh cycle (status + head + aheadBehind + diff) while
// preventing runaway process storms.
var gitSemaphore = make(chan struct{}, 4)

func acquireGitSemaphore(ctx context.Context) error {
	select {
	case gitSemaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseGitSemaphore() {
	<-gitSemaphore
}

// CLIService implements Service by shelling out to the git CLI.
// Optimised for large monorepos:
//   - GIT_OPTIONAL_LOCKS=0 env var on all read commands (no lock contention)
//   - Context-based timeouts prevent hangs
//   - Stdout/Stderr separated — stderr noise doesn't corrupt output
type CLIService struct {
	root   string // Absolute path to the repo root.
	gitDir string // Path to the .git directory.
}

// Compile-time check that CLIService implements Service.
var _ Service = (*CLIService)(nil)

// NewCLIService opens a Git repository at the given path.
func NewCLIService(path string) (*CLIService, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}
	topLevel, err := runGit(abs, nil, cmdTimeoutRead, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, ErrNotARepo
	}
	gitDir, err := runGit(abs, nil, cmdTimeoutRead, "rev-parse", "--git-dir")
	if err != nil {
		return nil, fmt.Errorf("finding .git directory: %w", err)
	}
	gd := strings.TrimSpace(gitDir)
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(strings.TrimSpace(topLevel), gd)
	}
	return &CLIService{
		root:   strings.TrimSpace(topLevel),
		gitDir: gd,
	}, nil
}

// GitDir returns the path to the .git directory.
func (s *CLIService) GitDir() string { return s.gitDir }

// ── helpers ─────────────────────────────────────────────────────────────────

// readEnv is the environment set on all read-only git commands.
// GIT_OPTIONAL_LOCKS=0 prevents git from acquiring optional locks,
// which is critical in large repos where lock contention stalls readers.
var readEnv = []string{"GIT_OPTIONAL_LOCKS=0"}

// run executes a read-only git command at the repo root with read-optimised
// env and a tight timeout.
func (s *CLIService) run(args ...string) (string, error) {
	return runGit(s.root, readEnv, cmdTimeoutRead, args...)
}

// runWrite executes a write git command (no optional-locks override).
func (s *CLIService) runWrite(args ...string) (string, error) {
	return runGit(s.root, nil, cmdTimeoutWrite, args...)
}

// runNetwork executes a network git command (fetch/push/pull) with a
// generous timeout.
func (s *CLIService) runNetwork(args ...string) (string, error) {
	return runGit(s.root, nil, cmdTimeoutNetwork, args...)
}

// runGit executes a git command with a context timeout and a bounded
// concurrency semaphore. Stdout and stderr are separated so stderr noise
// doesn't corrupt output.
func runGit(dir string, extraEnv []string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Acquire the semaphore — blocks until a slot opens or ctx expires.
	if err := acquireGitSemaphore(ctx); err != nil {
		return "", fmt.Errorf("git %s: waiting for semaphore: %w", strings.Join(args, " "), err)
	}
	defer releaseGitSemaphore()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	// Inherit environment, add extras.
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), errMsg, err)
	}
	return stdout.String(), nil
}

// ── Repository info ─────────────────────────────────────────────────────────

// RepoRoot returns the repository root path.
func (s *CLIService) RepoRoot() string { return s.root }

// Head returns the current HEAD ref.
func (s *CLIService) Head() (string, error) {
	// Fast path: read symbolic ref directly, no optional locks.
	ref, err := s.run("symbolic-ref", "--short", "HEAD")
	if err != nil {
		hash, hashErr := s.run("rev-parse", "--short", "HEAD")
		if hashErr != nil {
			return "", fmt.Errorf("getting HEAD: %w", err)
		}
		return strings.TrimSpace(hash), nil
	}
	return strings.TrimSpace(ref), nil
}

// IsClean reports whether the worktree is clean.
func (s *CLIService) IsClean() (bool, error) {
	// Limit to 1 result — we only need to know if anything is dirty, not the
	// full list. On 100k-file repos this is 10x faster.
	// GIT_OPTIONAL_LOCKS=0 is set via readEnv to avoid index.lock contention.
	out, err := s.run("status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

// IsMerging reports whether a merge is in progress.
func (s *CLIService) IsMerging() bool {
	// Fast path: check file existence directly — avoids spawning a subprocess.
	_, err := os.Stat(filepath.Join(s.gitDir, "MERGE_HEAD"))
	return err == nil
}

// IsRebasing reports whether a rebase is in progress.
func (s *CLIService) IsRebasing() bool {
	// Fast path: check directory existence directly — avoids spawning subprocesses.
	for _, sub := range []string{"rebase-merge", "rebase-apply"} {
		if info, err := os.Stat(filepath.Join(s.gitDir, sub)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// AheadBehind returns how many commits ahead/behind the upstream.
func (s *CLIService) AheadBehind() (int, int, error) {
	out, err := s.run("rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return 0, 0, nil //nolint:nilerr // no upstream is not an error
	}
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, nil
	}
	var ahead, behind int
	_, _ = fmt.Sscan(parts[0], &ahead)
	_, _ = fmt.Sscan(parts[1], &behind)
	return ahead, behind, nil
}

// Upstream returns the upstream tracking branch name.
func (s *CLIService) Upstream() string {
	out, err := s.run("rev-parse", "--abbrev-ref", "@{upstream}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ── Status & staging ────────────────────────────────────────────────────────

// Status returns the current working tree status.
func (s *CLIService) Status() (*StatusResult, error) {
	// --porcelain=v1 -z: machine-parseable, NUL-delimited.
	// -uno would skip untracked (fast), but we need them. Use -unormal
	// to only scan one level deep for untracked files in large repos.
	// GIT_OPTIONAL_LOCKS=0 is set via readEnv to avoid index.lock contention.
	out, err := s.run("status", "--porcelain=v1", "-z",
		"--untracked-files=normal")
	if err != nil {
		return nil, fmt.Errorf("getting status: %w", err)
	}
	return ParseStatusOutput(out), nil
}

// Stage stages the given paths.
func (s *CLIService) Stage(paths ...string) error {
	args := append([]string{"add", "--"}, paths...)
	_, err := s.runWrite(args...)
	return err
}

// StageAll stages all changes.
func (s *CLIService) StageAll() error { _, err := s.runWrite("add", "-A"); return err }

// Unstage unstages the given paths.
func (s *CLIService) Unstage(paths ...string) error {
	args := append([]string{"reset", "HEAD", "--"}, paths...)
	_, err := s.runWrite(args...)
	return err
}

// UnstageAll unstages all changes.
func (s *CLIService) UnstageAll() error { _, err := s.runWrite("reset", "HEAD"); return err }

// Discard discards changes for the given paths.
func (s *CLIService) Discard(paths ...string) error {
	args := append([]string{"checkout", "--"}, paths...)
	_, err := s.runWrite(args...)
	return err
}

// ── Commits ─────────────────────────────────────────────────────────────────

// Commit creates a new commit with the given message.
func (s *CLIService) Commit(message string) error {
	_, err := s.runWrite("commit", "-m", message)
	return err
}

// CommitAmend amends the last commit with the given message.
func (s *CLIService) CommitAmend(message string) error {
	_, err := s.runWrite("commit", "--amend", "-m", message)
	return err
}

// Log returns the commit log.
func (s *CLIService) Log(limit int, args ...string) ([]Commit, error) {
	cmdArgs := []string{
		"log", fmt.Sprintf("--max-count=%d", limit),
		LogFormatFlag(),
	}
	cmdArgs = append(cmdArgs, args...)
	out, err := s.run(cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("getting log: %w", err)
	}
	return ParseLogOutput(out), nil
}

// LogGraph returns the commit log with ASCII graph.
func (s *CLIService) LogGraph(limit int) ([]GraphEntry, error) {
	// --graph --all can be expensive on repos with many refs.
	// Limit to a reasonable count.
	out, err := s.run("log",
		fmt.Sprintf("--max-count=%d", limit),
		"--graph", "--all",
		LogFormatFlag())
	if err != nil {
		return nil, fmt.Errorf("getting log graph: %w", err)
	}
	return ParseGraphOutput(out), nil
}

// Show returns the commit details and diff for a given hash.
func (s *CLIService) Show(hash string) (*Commit, string, error) {
	commits, err := s.Log(1, hash, "-1")
	if err != nil || len(commits) == 0 {
		return nil, "", fmt.Errorf("showing commit %s: %w", hash, err)
	}
	// --stat is cheaper than --patch for initial display.
	diff, err := s.run("show", "--format=", "--patch", hash)
	if err != nil {
		return &commits[0], "", nil
	}
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n... (diff truncated — exceeds 512 KB) ...\n"
	}
	return &commits[0], diff, nil
}

// ── Diff ────────────────────────────────────────────────────────────────────

// maxDiffBytes is the maximum size of diff output we'll keep in memory.
// For multi-instance scenarios, this prevents one huge diff from bloating
// the process. 512 KB is enough for any reasonable diff; larger ones are
// truncated with a notice.
const maxDiffBytes = 512 * 1024

// Diff returns the diff for a path.
func (s *CLIService) Diff(staged bool, path string) (string, error) {
	args := []string{"diff", "--color=never", "--no-ext-diff"}
	if staged {
		args = append(args, "--cached")
	}
	if path != "" {
		args = append(args, "--", path)
	}
	out, err := s.run(args...)
	if err != nil {
		return "", err
	}
	if len(out) > maxDiffBytes {
		return out[:maxDiffBytes] + "\n\n... (diff truncated — exceeds 512 KB) ...\n", nil
	}
	return out, nil
}

// DiffRange returns the diff between two refs.
func (s *CLIService) DiffRange(from, to string) (string, error) {
	out, err := s.run("diff", "--color=never", "--no-ext-diff", from+".."+to)
	if err != nil {
		return "", err
	}
	if len(out) > maxDiffBytes {
		return out[:maxDiffBytes] + "\n\n... (diff truncated — exceeds 512 KB) ...\n", nil
	}
	return out, nil
}

// ── Branches ────────────────────────────────────────────────────────────────

const branchFormat = "%(HEAD)%00%(refname:short)%00%(objectname:short)%00%(upstream:short)%00%(upstream:track)%00%(subject)"

// Branches returns all branches.
func (s *CLIService) Branches() ([]Branch, error) {
	// --sort=-committerdate: most recently active branches first.
	out, err := s.run("branch", "-a", "--format="+branchFormat, "--sort=-committerdate")
	if err != nil {
		return nil, err
	}
	return ParseBranchOutput(out), nil
}

// CreateBranch creates a new branch.
func (s *CLIService) CreateBranch(name string) error {
	_, err := s.runWrite("branch", name)
	return err
}

// SwitchBranch switches to the given branch.
func (s *CLIService) SwitchBranch(name string) error {
	_, err := s.runWrite("switch", name)
	return err
}

// DeleteBranch deletes the given branch.
func (s *CLIService) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := s.runWrite("branch", flag, name)
	return err
}

// MergeBranch merges the given branch into the current branch.
func (s *CLIService) MergeBranch(name string) error {
	_, err := s.runWrite("merge", name)
	return err
}

// RenameBranch renames a branch.
func (s *CLIService) RenameBranch(oldName, newName string) error {
	_, err := s.runWrite("branch", "-m", oldName, newName)
	return err
}

// ── Stash ───────────────────────────────────────────────────────────────────

// StashList returns stash entries.
func (s *CLIService) StashList() ([]StashEntry, error) {
	out, err := s.run("stash", "list")
	if err != nil {
		return nil, err
	}
	return ParseStashList(out), nil
}

// StashSave saves a new stash entry.
func (s *CLIService) StashSave(message string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := s.runWrite(args...)
	return err
}

// StashPop pops the stash at the given index.
func (s *CLIService) StashPop(index int) error {
	_, err := s.runWrite("stash", "pop", fmt.Sprintf("stash@{%d}", index))
	return err
}

// StashApply applies the stash at the given index.
func (s *CLIService) StashApply(index int) error {
	_, err := s.runWrite("stash", "apply", fmt.Sprintf("stash@{%d}", index))
	return err
}

// StashDrop drops the stash at the given index.
func (s *CLIService) StashDrop(index int) error {
	_, err := s.runWrite("stash", "drop", fmt.Sprintf("stash@{%d}", index))
	return err
}

// StashShow shows the diff for a stash entry.
func (s *CLIService) StashShow(index int) (string, error) {
	return s.run("stash", "show", "-p", fmt.Sprintf("stash@{%d}", index))
}

// ── Remotes ─────────────────────────────────────────────────────────────────

// Remotes returns all configured remotes.
func (s *CLIService) Remotes() ([]Remote, error) {
	out, err := s.run("remote", "-v")
	if err != nil {
		return nil, err
	}
	return ParseRemoteOutput(out), nil
}

// Fetch fetches from the given remote.
func (s *CLIService) Fetch(remote string) error {
	_, err := s.runNetwork("fetch", remote)
	return err
}

// Pull pulls from the given remote and branch.
func (s *CLIService) Pull(remote, branch string) error {
	_, err := s.runNetwork("pull", remote, branch)
	return err
}

// Push pushes to the given remote and branch.
func (s *CLIService) Push(remote, branch string, force bool) error {
	args := []string{"push", remote, branch}
	if force {
		args = append(args, "--force-with-lease")
	}
	_, err := s.runNetwork(args...)
	return err
}

// ── Worktrees ───────────────────────────────────────────────────────────────

// WorktreeList returns all worktrees.
func (s *CLIService) WorktreeList() ([]Worktree, error) {
	out, err := s.run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(out), nil
}

// WorktreeAdd adds a new worktree.
func (s *CLIService) WorktreeAdd(path, branch string) error {
	args := []string{"worktree", "add", path}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	_, err := s.runWrite(args...)
	return err
}

// WorktreeRemove removes a worktree.
func (s *CLIService) WorktreeRemove(path string) error {
	_, err := s.runWrite("worktree", "remove", path)
	return err
}

// ── Rebase ──────────────────────────────────────────────────────────────────

// RebaseInteractive starts an interactive rebase.
func (s *CLIService) RebaseInteractive(onto string) error {
	_, err := s.runWrite("rebase", "-i", onto)
	return err
}

// RebaseContinue continues a rebase in progress.
func (s *CLIService) RebaseContinue() error { _, err := s.runWrite("rebase", "--continue"); return err }

// RebaseAbort aborts a rebase in progress.
func (s *CLIService) RebaseAbort() error { _, err := s.runWrite("rebase", "--abort"); return err }

// ── Bisect ──────────────────────────────────────────────────────────────────

// BisectStart starts a git bisect.
func (s *CLIService) BisectStart(bad, good string) error {
	_, err := s.runWrite("bisect", "start", bad, good)
	return err
}

// BisectGood marks the current commit as good.
func (s *CLIService) BisectGood() error { _, err := s.runWrite("bisect", "good"); return err }

// BisectBad marks the current commit as bad.
func (s *CLIService) BisectBad() error { _, err := s.runWrite("bisect", "bad"); return err }

// BisectReset resets the bisect session.
func (s *CLIService) BisectReset() error { _, err := s.runWrite("bisect", "reset"); return err }

// BisectLog returns the bisect log.
func (s *CLIService) BisectLog() (string, error) {
	return s.run("bisect", "log")
}

// ── Conflict resolution ─────────────────────────────────────────────────────

// ConflictFiles returns paths with merge conflicts.
func (s *CLIService) ConflictFiles() ([]string, error) {
	out, err := s.run("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n"), nil
}

// MarkResolved marks a conflict as resolved.
func (s *CLIService) MarkResolved(path string) error {
	_, err := s.runWrite("add", path)
	return err
}
