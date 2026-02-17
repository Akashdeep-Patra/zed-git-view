package git

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ── Log / commit parsing ────────────────────────────────────────────────────

const (
	logFormat    = "%H%x00%h%x00%an%x00%ae%x00%at%x00%ar%x00%s%x00%b%x00%P%x00%D"
	logSeparator = "%x01"
)

// LogFormatFlag returns the --format flag for git log.
func LogFormatFlag() string {
	return fmt.Sprintf("--format=%s%s", logFormat, logSeparator)
}

// ParseLogOutput parses the raw output of git log using our custom format.
// Optimised: uses IndexByte scanning instead of Split to avoid allocating
// a large []string for repos with thousands of commits.
func ParseLogOutput(out string) []Commit {
	if len(out) == 0 {
		return nil
	}
	// Estimate capacity: typical commit entry is ~200 bytes.
	est := len(out) / 200
	if est < 8 {
		est = 8
	}
	commits := make([]Commit, 0, est)

	for len(out) > 0 {
		idx := strings.IndexByte(out, '\x01')
		var entry string
		if idx < 0 {
			entry = out
			out = ""
		} else {
			entry = out[:idx]
			out = out[idx+1:]
		}
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if c, ok := parseCommitEntry(entry); ok {
			commits = append(commits, c)
		}
	}
	return commits
}

func parseCommitEntry(entry string) (Commit, bool) {
	parts := strings.SplitN(entry, "\x00", 10)
	if len(parts) < 10 {
		return Commit{}, false
	}
	ts, _ := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
	c := Commit{
		Hash:        strings.TrimSpace(parts[0]),
		ShortHash:   strings.TrimSpace(parts[1]),
		Author:      strings.TrimSpace(parts[2]),
		AuthorEmail: strings.TrimSpace(parts[3]),
		Date:        time.Unix(ts, 0),
		RelDate:     strings.TrimSpace(parts[5]),
		Subject:     strings.TrimSpace(parts[6]),
		Body:        strings.TrimSpace(parts[7]),
	}
	if p := strings.TrimSpace(parts[8]); p != "" {
		c.Parents = strings.Fields(p)
	}
	if r := strings.TrimSpace(parts[9]); r != "" {
		c.Refs = ParseRefs(r)
	}
	return c, true
}

// ParseRefs parses the %D decoration string into typed Ref values.
func ParseRefs(raw string) []Ref {
	refs := make([]Ref, 0, 4)
	for _, r := range strings.Split(raw, ", ") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		ref := Ref{Name: r}
		switch {
		case r == "HEAD":
			ref.Type = RefHead
		case strings.HasPrefix(r, "HEAD -> "):
			ref.Name = strings.TrimPrefix(r, "HEAD -> ")
			ref.Type = RefHead
		case strings.HasPrefix(r, "tag: "):
			ref.Name = strings.TrimPrefix(r, "tag: ")
			ref.Type = RefTag
		case strings.Contains(r, "/"):
			ref.Type = RefRemoteBranch
			parts := strings.SplitN(r, "/", 2)
			ref.Remote = parts[0]
			ref.Name = parts[1]
		default:
			ref.Type = RefBranch
		}
		refs = append(refs, ref)
	}
	return refs
}

// ── Status parsing ──────────────────────────────────────────────────────────

// ParseStatusOutput parses `git status --porcelain=v1 -z`.
// NUL-delimited scanning avoids allocating a massive []string for repos
// with thousands of changed files.
func ParseStatusOutput(out string) *StatusResult {
	result := &StatusResult{}
	if len(out) == 0 {
		return result
	}

	// Pre-allocate with reasonable defaults for monorepos.
	result.Staged = make([]FileStatus, 0, 32)
	result.Unstaged = make([]FileStatus, 0, 32)
	result.Untracked = make([]FileStatus, 0, 16)

	// Scan NUL-separated entries without strings.Split.
	for len(out) > 0 {
		nul := strings.IndexByte(out, '\x00')
		var entry string
		if nul < 0 {
			entry = out
			out = ""
		} else {
			entry = out[:nul]
			out = out[nul+1:]
		}
		if len(entry) < 4 {
			continue
		}

		staging := StatusCode(entry[0])
		worktree := StatusCode(entry[1])
		path := entry[3:]

		fs := FileStatus{Staging: staging, Worktree: worktree, Path: path}

		// Renames/copies have an extra NUL-separated entry for the original path.
		if staging == StatusRenamed || staging == StatusCopied ||
			worktree == StatusRenamed || worktree == StatusCopied {
			nul2 := strings.IndexByte(out, '\x00')
			if nul2 < 0 {
				fs.OrigPath = out
				out = ""
			} else {
				fs.OrigPath = out[:nul2]
				out = out[nul2+1:]
			}
		}

		if staging == StatusUntracked && worktree == StatusUntracked {
			result.Untracked = append(result.Untracked, fs)
			continue
		}

		if staging == StatusUnmerged || worktree == StatusUnmerged ||
			(staging == StatusAdded && worktree == StatusAdded) ||
			(staging == StatusDeleted && worktree == StatusDeleted) {
			result.Conflicts = append(result.Conflicts, fs)
			continue
		}

		if staging != StatusUnmodified && staging != StatusUntracked {
			staged := fs
			staged.IsStaged = true
			result.Staged = append(result.Staged, staged)
		}
		if worktree != StatusUnmodified && worktree != StatusUntracked {
			result.Unstaged = append(result.Unstaged, fs)
		}
	}
	return result
}

// ── Branch parsing ──────────────────────────────────────────────────────────

// ParseBranchOutput parses `git branch -a --format=...`.
func ParseBranchOutput(out string) []Branch {
	if len(out) == 0 {
		return nil
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	branches := make([]Branch, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\x00", 6)
		if len(parts) < 6 {
			continue
		}
		b := Branch{
			IsCurrent: strings.TrimSpace(parts[0]) == "*",
			Name:      strings.TrimSpace(parts[1]),
			Hash:      strings.TrimSpace(parts[2]),
			Upstream:  strings.TrimSpace(parts[3]),
			Subject:   strings.TrimSpace(parts[5]),
		}
		if ab := strings.TrimSpace(parts[4]); ab != "" && ab != "gone" {
			_, _ = fmt.Sscanf(ab, "[ahead %d, behind %d]", &b.Ahead, &b.Behind)
			if b.Ahead == 0 {
				_, _ = fmt.Sscanf(ab, "[ahead %d]", &b.Ahead)
			}
			if b.Behind == 0 {
				_, _ = fmt.Sscanf(ab, "[behind %d]", &b.Behind)
			}
		}
		b.IsRemote = strings.HasPrefix(b.Name, "remotes/")
		if b.IsRemote {
			b.Name = strings.TrimPrefix(b.Name, "remotes/")
		}
		branches = append(branches, b)
	}
	return branches
}

// ── Stash parsing ───────────────────────────────────────────────────────────

// ParseStashList parses `git stash list`.
func ParseStashList(out string) []StashEntry {
	if len(out) == 0 {
		return nil
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	entries := make([]StashEntry, 0, len(lines))
	for _, line := range lines {
		var idx int
		if _, err := fmt.Sscanf(line, "stash@{%d}", &idx); err != nil {
			continue
		}
		msg := line
		if colonIdx := strings.Index(line, ": "); colonIdx != -1 {
			rest := line[colonIdx+2:]
			if secondColon := strings.Index(rest, ": "); secondColon != -1 {
				msg = rest[secondColon+2:]
			} else {
				msg = rest
			}
		}
		branch := ""
		if strings.Contains(line, "On ") {
			parts := strings.SplitN(line, "On ", 2)
			if len(parts) == 2 {
				if colonIdx := strings.Index(parts[1], ":"); colonIdx != -1 {
					branch = parts[1][:colonIdx]
				}
			}
		}
		entries = append(entries, StashEntry{Index: idx, Message: msg, Branch: branch})
	}
	return entries
}

// ── Remote parsing ──────────────────────────────────────────────────────────

// ParseRemoteOutput parses `git remote -v`.
func ParseRemoteOutput(out string) []Remote {
	if len(out) == 0 {
		return nil
	}
	seen := map[string]*Remote{}
	var order []string
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		url := fields[1]
		kind := strings.Trim(fields[2], "()")
		r, ok := seen[name]
		if !ok {
			r = &Remote{Name: name}
			seen[name] = r
			order = append(order, name)
		}
		switch kind {
		case "fetch":
			r.FetchURL = url
		case "push":
			r.PushURL = url
		}
	}
	remotes := make([]Remote, 0, len(order))
	for _, name := range order {
		remotes = append(remotes, *seen[name])
	}
	return remotes
}

// ── Worktree parsing ────────────────────────────────────────────────────────

// ParseWorktreeList parses `git worktree list --porcelain`.
func ParseWorktreeList(out string) []Worktree {
	if len(out) == 0 {
		return nil
	}
	var wts []Worktree
	var cur Worktree
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if cur.Path != "" {
				wts = append(wts, cur)
			}
			cur = Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(line, "branch ")
		case line == "bare":
			cur.Bare = true
		}
	}
	if cur.Path != "" {
		wts = append(wts, cur)
	}
	return wts
}

// ── Graph parsing ───────────────────────────────────────────────────────────

// ParseGraphOutput parses `git log --graph` with our custom format.
func ParseGraphOutput(out string) []GraphEntry {
	if len(out) == 0 {
		return nil
	}
	lines := strings.Split(out, "\n")
	entries := make([]GraphEntry, 0, len(lines)/2)
	var commitBuf strings.Builder
	var graphBuf string

	flush := func() {
		if commitBuf.Len() == 0 {
			return
		}
		raw := strings.TrimSpace(commitBuf.String())
		commitBuf.Reset()
		if c, ok := parseCommitEntry(raw); ok {
			entries = append(entries, GraphEntry{Graph: graphBuf, Commit: &c})
		}
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		graphEnd := findGraphEnd(line)
		graph := line[:graphEnd]
		content := line[graphEnd:]

		switch {
		case strings.Contains(content, "\x00"):
			flush()
			graphBuf = graph
			commitBuf.WriteString(content)
		case commitBuf.Len() > 0:
			commitBuf.WriteString(content)
		default:
			entries = append(entries, GraphEntry{Graph: graph + content})
		}
	}
	flush()
	return entries
}

func findGraphEnd(line string) int {
	for i, ch := range line {
		if ch != '*' && ch != '|' && ch != '/' && ch != '\\' && ch != '_' && ch != ' ' {
			return i
		}
	}
	return len(line)
}
