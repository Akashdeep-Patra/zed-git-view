package git

import (
	"sync"
	"time"
)

// CachedService wraps a Service implementation with a TTL-based cache for
// expensive read operations. Write operations (Stage, Commit, etc.)
// automatically invalidate the cache so the next read is fresh.
//
// This is critical for monorepo performance: multiple views and the status
// bar all request overlapping data (Status, Head, AheadBehind, etc.)
// within the same refresh cycle. Without caching, a single refresh event
// could spawn 15+ git subprocesses. With caching, it spawns ~5.
//
// The cache is bounded by maxCacheEntries to prevent unbounded memory
// growth across long-running sessions or multiple instances.
type CachedService struct {
	inner Service
	ttl   time.Duration

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// maxCacheEntries caps the number of entries in the cache. When exceeded,
// the entire cache is flushed (simple but effective — the TTL is short
// so this only happens if something is wrong).
const maxCacheEntries = 64

type cacheEntry struct {
	val    interface{}
	err    error
	expiry time.Time
}

// Compile-time check.
var _ Service = (*CachedService)(nil)

// NewCachedService wraps an existing Service with a TTL cache.
// Recommended TTL: 1-2 seconds. This ensures that within a single
// refresh cycle (which triggers multiple git queries), each query
// only hits git once.
func NewCachedService(inner Service, ttl time.Duration) *CachedService {
	return &CachedService{
		inner: inner,
		ttl:   ttl,
		cache: make(map[string]cacheEntry, 16),
	}
}

// Invalidate clears all cached entries. Called after any write operation.
func (c *CachedService) Invalidate() {
	c.mu.Lock()
	c.cache = make(map[string]cacheEntry, 16)
	c.mu.Unlock()
}

func (c *CachedService) get(key string) (val interface{}, ok bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, found := c.cache[key]
	if !found || time.Now().After(e.expiry) {
		return nil, false, nil
	}
	return e.val, true, e.err
}

func (c *CachedService) set(key string, val interface{}, err error) {
	c.mu.Lock()
	// Evict expired entries if the cache is getting large.
	if len(c.cache) >= maxCacheEntries {
		now := time.Now()
		for k, e := range c.cache {
			if now.After(e.expiry) {
				delete(c.cache, k)
			}
		}
		// If still over limit after eviction, flush entirely.
		if len(c.cache) >= maxCacheEntries {
			c.cache = make(map[string]cacheEntry, 16)
		}
	}
	c.cache[key] = cacheEntry{val: val, err: err, expiry: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

// invalidateAndReturn is a helper for write methods.
func (c *CachedService) invalidateAndReturn(err error) error {
	if err == nil {
		c.Invalidate()
	}
	return err
}

// ── Repository info (cached reads) ──────────────────────────────────────────

// RepoRoot delegates to the inner service.
func (c *CachedService) RepoRoot() string { return c.inner.RepoRoot() }

// GitDir delegates to the inner service.
func (c *CachedService) GitDir() string { return c.inner.GitDir() }

// Head returns the current HEAD ref (cached).
func (c *CachedService) Head() (string, error) {
	if v, ok, err := c.get("head"); ok {
		return v.(string), err
	}
	v, err := c.inner.Head()
	c.set("head", v, err)
	return v, err
}

// IsClean reports whether the worktree is clean (cached).
func (c *CachedService) IsClean() (bool, error) {
	if v, ok, err := c.get("isclean"); ok {
		return v.(bool), err
	}
	v, err := c.inner.IsClean()
	c.set("isclean", v, err)
	return v, err
}

// IsMerging delegates to the inner service (cached).
func (c *CachedService) IsMerging() bool {
	if v, ok, _ := c.get("ismerging"); ok {
		return v.(bool)
	}
	v := c.inner.IsMerging()
	c.set("ismerging", v, nil)
	return v
}

// IsRebasing delegates to the inner service (cached).
func (c *CachedService) IsRebasing() bool {
	if v, ok, _ := c.get("isrebasing"); ok {
		return v.(bool)
	}
	v := c.inner.IsRebasing()
	c.set("isrebasing", v, nil)
	return v
}

// AheadBehind delegates to the inner service (cached).
func (c *CachedService) AheadBehind() (int, int, error) {
	type ab struct{ a, b int }
	if v, ok, err := c.get("aheadbehind"); ok {
		r := v.(ab)
		return r.a, r.b, err
	}
	a, b, err := c.inner.AheadBehind()
	c.set("aheadbehind", ab{a, b}, err)
	return a, b, err
}

// Upstream delegates to the inner service (cached).
func (c *CachedService) Upstream() string {
	if v, ok, _ := c.get("upstream"); ok {
		return v.(string)
	}
	v := c.inner.Upstream()
	c.set("upstream", v, nil)
	return v
}

// ── Status (cached) ─────────────────────────────────────────────────────────

// Status delegates to the inner service (cached).
func (c *CachedService) Status() (*StatusResult, error) {
	if v, ok, err := c.get("status"); ok {
		return v.(*StatusResult), err
	}
	v, err := c.inner.Status()
	c.set("status", v, err)
	return v, err
}

// ── Write operations (invalidate cache) ─────────────────────────────────────

// Stage stages paths and invalidates the cache.
func (c *CachedService) Stage(paths ...string) error {
	return c.invalidateAndReturn(c.inner.Stage(paths...))
}

// StageAll stages all changes and invalidates the cache.
func (c *CachedService) StageAll() error {
	return c.invalidateAndReturn(c.inner.StageAll())
}

// Unstage unstages paths and invalidates the cache.
func (c *CachedService) Unstage(paths ...string) error {
	return c.invalidateAndReturn(c.inner.Unstage(paths...))
}

// UnstageAll unstages all paths and invalidates the cache.
func (c *CachedService) UnstageAll() error {
	return c.invalidateAndReturn(c.inner.UnstageAll())
}

// Discard discards changes in paths and invalidates the cache.
func (c *CachedService) Discard(paths ...string) error {
	return c.invalidateAndReturn(c.inner.Discard(paths...))
}

// Commit creates a commit and invalidates the cache.
func (c *CachedService) Commit(message string) error {
	return c.invalidateAndReturn(c.inner.Commit(message))
}

// CommitAmend amends the last commit and invalidates the cache.
func (c *CachedService) CommitAmend(message string) error {
	return c.invalidateAndReturn(c.inner.CommitAmend(message))
}

// ── Log (not cached — already limited by max-count) ─────────────────────────

// Log delegates to the inner service (not cached).
func (c *CachedService) Log(limit int, args ...string) ([]Commit, error) {
	return c.inner.Log(limit, args...)
}

// LogGraph delegates to the inner service (not cached).
func (c *CachedService) LogGraph(limit int) ([]GraphEntry, error) {
	return c.inner.LogGraph(limit)
}

// Show delegates to the inner service (not cached).
func (c *CachedService) Show(hash string) (*Commit, string, error) {
	return c.inner.Show(hash)
}

// ── Diff (not cached — content is large and changes per-file) ───────────────

// Diff delegates to the inner service (not cached).
func (c *CachedService) Diff(staged bool, path string) (string, error) {
	return c.inner.Diff(staged, path)
}

// DiffRange delegates to the inner service (not cached).
func (c *CachedService) DiffRange(from, to string) (string, error) {
	return c.inner.DiffRange(from, to)
}

// ── Branches (cached) ───────────────────────────────────────────────────────

// Branches delegates to the inner service (cached).
func (c *CachedService) Branches() ([]Branch, error) {
	if v, ok, err := c.get("branches"); ok {
		return v.([]Branch), err
	}
	v, err := c.inner.Branches()
	c.set("branches", v, err)
	return v, err
}

// CreateBranch creates a branch and invalidates the cache.
func (c *CachedService) CreateBranch(name string) error {
	return c.invalidateAndReturn(c.inner.CreateBranch(name))
}

// SwitchBranch switches to a branch and invalidates the cache.
func (c *CachedService) SwitchBranch(name string) error {
	return c.invalidateAndReturn(c.inner.SwitchBranch(name))
}

// DeleteBranch deletes a branch and invalidates the cache.
func (c *CachedService) DeleteBranch(name string, force bool) error {
	return c.invalidateAndReturn(c.inner.DeleteBranch(name, force))
}

// MergeBranch merges a branch and invalidates the cache.
func (c *CachedService) MergeBranch(name string) error {
	return c.invalidateAndReturn(c.inner.MergeBranch(name))
}

// RenameBranch renames a branch and invalidates the cache.
func (c *CachedService) RenameBranch(oldName, newName string) error {
	return c.invalidateAndReturn(c.inner.RenameBranch(oldName, newName))
}

// ── Stash (cached list, invalidate on mutation) ─────────────────────────────

// StashList delegates to the inner service (cached).
func (c *CachedService) StashList() ([]StashEntry, error) {
	if v, ok, err := c.get("stashlist"); ok {
		return v.([]StashEntry), err
	}
	v, err := c.inner.StashList()
	c.set("stashlist", v, err)
	return v, err
}

// StashSave saves to stash and invalidates the cache.
func (c *CachedService) StashSave(message string) error {
	return c.invalidateAndReturn(c.inner.StashSave(message))
}

// StashPop pops a stash entry and invalidates the cache.
func (c *CachedService) StashPop(index int) error {
	return c.invalidateAndReturn(c.inner.StashPop(index))
}

// StashApply applies a stash entry and invalidates the cache.
func (c *CachedService) StashApply(index int) error {
	return c.invalidateAndReturn(c.inner.StashApply(index))
}

// StashDrop drops a stash entry and invalidates the cache.
func (c *CachedService) StashDrop(index int) error {
	return c.invalidateAndReturn(c.inner.StashDrop(index))
}

// StashShow delegates to the inner service (not cached).
func (c *CachedService) StashShow(index int) (string, error) {
	return c.inner.StashShow(index)
}

// ── Remotes (cached) ────────────────────────────────────────────────────────

// Remotes delegates to the inner service (cached).
func (c *CachedService) Remotes() ([]Remote, error) {
	if v, ok, err := c.get("remotes"); ok {
		return v.([]Remote), err
	}
	v, err := c.inner.Remotes()
	c.set("remotes", v, err)
	return v, err
}

// Fetch fetches from remote and invalidates the cache.
func (c *CachedService) Fetch(remote string) error {
	return c.invalidateAndReturn(c.inner.Fetch(remote))
}

// Pull pulls from remote and invalidates the cache.
func (c *CachedService) Pull(remote, branch string) error {
	return c.invalidateAndReturn(c.inner.Pull(remote, branch))
}

// Push pushes to remote and invalidates the cache.
func (c *CachedService) Push(remote, branch string, force bool) error {
	return c.invalidateAndReturn(c.inner.Push(remote, branch, force))
}

// ── Worktrees ───────────────────────────────────────────────────────────────

// WorktreeList delegates to the inner service (cached).
func (c *CachedService) WorktreeList() ([]Worktree, error) {
	if v, ok, err := c.get("worktrees"); ok {
		return v.([]Worktree), err
	}
	v, err := c.inner.WorktreeList()
	c.set("worktrees", v, err)
	return v, err
}

// WorktreeAdd adds a worktree and invalidates the cache.
func (c *CachedService) WorktreeAdd(path, branch string) error {
	return c.invalidateAndReturn(c.inner.WorktreeAdd(path, branch))
}

// WorktreeRemove removes a worktree and invalidates the cache.
func (c *CachedService) WorktreeRemove(path string) error {
	return c.invalidateAndReturn(c.inner.WorktreeRemove(path))
}

// ── Rebase (write-only, always invalidates) ─────────────────────────────────

// RebaseInteractive starts interactive rebase and invalidates the cache.
func (c *CachedService) RebaseInteractive(onto string) error {
	return c.invalidateAndReturn(c.inner.RebaseInteractive(onto))
}

// RebaseContinue continues rebase and invalidates the cache.
func (c *CachedService) RebaseContinue() error {
	return c.invalidateAndReturn(c.inner.RebaseContinue())
}

// RebaseAbort aborts rebase and invalidates the cache.
func (c *CachedService) RebaseAbort() error {
	return c.invalidateAndReturn(c.inner.RebaseAbort())
}

// ── Bisect ──────────────────────────────────────────────────────────────────

// BisectStart starts bisect and invalidates the cache.
func (c *CachedService) BisectStart(bad, good string) error {
	return c.invalidateAndReturn(c.inner.BisectStart(bad, good))
}

// BisectGood marks current commit as good and invalidates the cache.
func (c *CachedService) BisectGood() error {
	return c.invalidateAndReturn(c.inner.BisectGood())
}

// BisectBad marks current commit as bad and invalidates the cache.
func (c *CachedService) BisectBad() error {
	return c.invalidateAndReturn(c.inner.BisectBad())
}

// BisectReset resets bisect and invalidates the cache.
func (c *CachedService) BisectReset() error {
	return c.invalidateAndReturn(c.inner.BisectReset())
}

// BisectLog delegates to the inner service (not cached).
func (c *CachedService) BisectLog() (string, error) {
	return c.inner.BisectLog()
}

// ── Conflict resolution ─────────────────────────────────────────────────────

// ConflictFiles delegates to the inner service (cached).
func (c *CachedService) ConflictFiles() ([]string, error) {
	if v, ok, err := c.get("conflicts"); ok {
		return v.([]string), err
	}
	v, err := c.inner.ConflictFiles()
	c.set("conflicts", v, err)
	return v, err
}

// MarkResolved marks a conflict as resolved and invalidates the cache.
func (c *CachedService) MarkResolved(path string) error {
	return c.invalidateAndReturn(c.inner.MarkResolved(path))
}
