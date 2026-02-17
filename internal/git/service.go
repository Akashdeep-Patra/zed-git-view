package git

// Service defines the contract for all Git operations.
// Every TUI view depends on this interface, never on exec.Command directly.
// This makes the application testable via mock implementations.
type Service interface {
	// ── Repository info ──────────────────────────────────────────────
	RepoRoot() string
	GitDir() string
	Head() (string, error)
	IsClean() (bool, error)
	IsMerging() bool
	IsRebasing() bool
	AheadBehind() (ahead, behind int, err error)
	Upstream() string

	// ── Status & staging ─────────────────────────────────────────────
	Status() (*StatusResult, error)
	Stage(paths ...string) error
	StageAll() error
	Unstage(paths ...string) error
	UnstageAll() error
	Discard(paths ...string) error

	// ── Commits ──────────────────────────────────────────────────────
	Commit(message string) error
	CommitAmend(message string) error
	Log(limit int, args ...string) ([]Commit, error)
	LogGraph(limit int) ([]GraphEntry, error)
	Show(hash string) (*Commit, string, error)

	// ── Diff ─────────────────────────────────────────────────────────
	Diff(staged bool, path string) (string, error)
	DiffRange(from, to string) (string, error)

	// ── Branches ─────────────────────────────────────────────────────
	Branches() ([]Branch, error)
	CreateBranch(name string) error
	SwitchBranch(name string) error
	DeleteBranch(name string, force bool) error
	MergeBranch(name string) error
	RenameBranch(oldName, newName string) error

	// ── Stash ────────────────────────────────────────────────────────
	StashList() ([]StashEntry, error)
	StashSave(message string) error
	StashPop(index int) error
	StashApply(index int) error
	StashDrop(index int) error
	StashShow(index int) (string, error)

	// ── Remotes ──────────────────────────────────────────────────────
	Remotes() ([]Remote, error)
	Fetch(remote string) error
	Pull(remote, branch string) error
	Push(remote, branch string, force bool) error

	// ── Worktrees ────────────────────────────────────────────────────
	WorktreeList() ([]Worktree, error)
	WorktreeAdd(path, branch string) error
	WorktreeRemove(path string) error

	// ── Rebase ───────────────────────────────────────────────────────
	RebaseInteractive(onto string) error
	RebaseContinue() error
	RebaseAbort() error

	// ── Bisect ───────────────────────────────────────────────────────
	BisectStart(bad, good string) error
	BisectGood() error
	BisectBad() error
	BisectReset() error
	BisectLog() (string, error)

	// ── Conflict resolution ──────────────────────────────────────────
	ConflictFiles() ([]string, error)
	MarkResolved(path string) error
}
