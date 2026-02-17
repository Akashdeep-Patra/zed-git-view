package git

import "time"

// StatusCode represents a single-character Git status indicator.
type StatusCode byte

// Git status codes as single-byte indicators.
const (
	StatusUnmodified  StatusCode = ' '
	StatusModified    StatusCode = 'M'
	StatusTypeChanged StatusCode = 'T'
	StatusAdded       StatusCode = 'A'
	StatusDeleted     StatusCode = 'D'
	StatusRenamed     StatusCode = 'R'
	StatusCopied      StatusCode = 'C'
	StatusUnmerged    StatusCode = 'U'
	StatusUntracked   StatusCode = '?'
	StatusIgnored     StatusCode = '!'
)

// String returns the single-character representation.
func (s StatusCode) String() string { return string(s) }

// Label returns a human-readable description of the status.
func (s StatusCode) Label() string {
	switch s {
	case StatusModified:
		return "Modified"
	case StatusTypeChanged:
		return "Type Changed"
	case StatusAdded:
		return "Added"
	case StatusDeleted:
		return "Deleted"
	case StatusRenamed:
		return "Renamed"
	case StatusCopied:
		return "Copied"
	case StatusUnmerged:
		return "Unmerged"
	case StatusUntracked:
		return "Untracked"
	case StatusIgnored:
		return "Ignored"
	default:
		return ""
	}
}

// FileStatus represents the status of a single file in the working tree or index.
type FileStatus struct {
	Staging  StatusCode
	Worktree StatusCode
	Path     string
	OrigPath string // Only set for renames/copies.
	IsStaged bool
}

// StatusResult holds the categorised status of the entire repository.
type StatusResult struct {
	Staged    []FileStatus
	Unstaged  []FileStatus
	Untracked []FileStatus
	Conflicts []FileStatus
}

// TotalCount returns the total number of files across all categories.
func (sr *StatusResult) TotalCount() int {
	return len(sr.Staged) + len(sr.Unstaged) + len(sr.Untracked) + len(sr.Conflicts)
}

// RefType classifies a Git reference.
type RefType int

// Git reference types.
const (
	RefBranch RefType = iota
	RefRemoteBranch
	RefTag
	RefHead
	RefStash
)

// Ref is a Git reference (branch, tag, HEAD, etc.).
type Ref struct {
	Name   string
	Type   RefType
	Remote string
}

// Commit represents a single Git commit.
type Commit struct {
	Hash        string
	ShortHash   string
	Author      string
	AuthorEmail string
	Date        time.Time
	RelDate     string
	Subject     string
	Body        string
	Parents     []string
	Refs        []Ref
}

// GraphEntry pairs a commit with its ASCII graph decoration.
type GraphEntry struct {
	Graph  string  // e.g. "* ", "| * "
	Commit *Commit // nil for graph-only lines (merge lines, etc.)
}

// Branch represents a local or remote branch.
type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	Upstream  string
	Hash      string
	Subject   string
	Ahead     int
	Behind    int
}

// StashEntry represents a single stash entry.
type StashEntry struct {
	Index   int
	Message string
	Branch  string
}

// Remote represents a configured Git remote.
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// Worktree represents a linked working tree.
type Worktree struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
}
