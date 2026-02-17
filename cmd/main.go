package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/Akashdeep-Patra/zed-git-view/internal/app"
	"github.com/Akashdeep-Patra/zed-git-view/internal/common"
	"github.com/Akashdeep-Patra/zed-git-view/internal/config"
	"github.com/Akashdeep-Patra/zed-git-view/internal/git"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui"
	"github.com/Akashdeep-Patra/zed-git-view/internal/ui/views"
	"github.com/Akashdeep-Patra/zed-git-view/internal/watcher"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Build-time variables injected via ldflags by GoReleaser / Taskfile.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	// ── Multi-instance resource tuning ──────────────────────────────
	//
	// When users run 5+ zgv instances across terminals, each Go runtime
	// defaults to GOMAXPROCS = NumCPU (e.g. 10 on an M1 Pro). That means
	// 5 × 10 = 50 OS threads competing for 10 cores, causing context-switch
	// overhead and latency spikes.
	//
	// A TUI app spends most of its time waiting for I/O (git subprocesses,
	// fsnotify, terminal input). 2 OS threads is plenty for the actual Go
	// work (render + message dispatch). The git subprocesses run externally
	// and aren't affected by GOMAXPROCS.
	//
	// If the user explicitly sets GOMAXPROCS, we respect that.
	if os.Getenv("GOMAXPROCS") == "" {
		maxProcs := 2
		if n := runtime.NumCPU(); n < maxProcs {
			maxProcs = n
		}
		runtime.GOMAXPROCS(maxProcs)
	}

	// Limit the GC target to 50 MB. For a TUI that should rarely exceed
	// 30 MB resident, this triggers the GC earlier and keeps RSS low —
	// critical when 5+ instances share the machine.
	debug.SetMemoryLimit(50 * 1024 * 1024) // 50 MiB
}

func main() {
	rootCmd := buildRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func buildRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "zgv",
		Short: "A modern TUI Git client for Zed IDE",
		Long: `zgv is a keyboard-first, terminal-based Git client designed to run
inside Zed's integrated terminal (or any terminal emulator).

It provides interactive views for status, log, diff, branches, stash,
remotes, rebase, conflict resolution, worktrees, and bisect — all from
a single TUI powered by Bubbletea.`,
		RunE:          runApp,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
	}

	rootCmd.SetVersionTemplate(fmt.Sprintf(
		"zgv %s\n  commit:  %s\n  built:   %s\n  go:      %s\n  os/arch: %s/%s\n",
		version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH,
	))

	rootCmd.AddCommand(buildVersionCmd())
	rootCmd.AddCommand(buildCompletionCmd())

	rootCmd.Flags().StringP("path", "p", ".", "Path to the git repository")

	return rootCmd
}

// buildVersionCmd creates the `zgv version` subcommand supporting --json.
func buildVersionCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			info := map[string]string{
				"version": version,
				"commit":  commit,
				"date":    date,
				"go":      runtime.Version(),
				"os":      runtime.GOOS,
				"arch":    runtime.GOARCH,
			}
			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			fmt.Printf("zgv %s\n", version)
			fmt.Printf("  commit:  %s\n", commit)
			fmt.Printf("  built:   %s\n", date)
			fmt.Printf("  go:      %s\n", runtime.Version())
			fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output version info as JSON")

	return cmd
}

// buildCompletionCmd creates the `zgv completion` subcommand for shell completions.
func buildCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for zgv.

Examples:
  # Bash (add to ~/.bashrc)
  zgv completion bash > /etc/bash_completion.d/zgv
  
  # Zsh (add to ~/.zshrc before compinit)
  zgv completion zsh > "${fpath[1]}/_zgv"
  
  # Fish
  zgv completion fish > ~/.config/fish/completions/zgv.fish
  
  # PowerShell
  zgv completion powershell > zgv.ps1`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	return cmd
}

func runApp(cmd *cobra.Command, _ []string) error {
	repoPath, _ := cmd.Flags().GetString("path")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cliSvc, err := git.NewCLIService(repoPath)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	// Wrap with a 2-second TTL cache to deduplicate git calls within a
	// single refresh cycle. Critical for monorepo performance.
	gitSvc := git.NewCachedService(cliSvc, 2*time.Second)

	styles := ui.DefaultStyles()

	viewMap := map[common.TabID]common.View{
		common.TabStatus:    views.NewStatusView(gitSvc, styles),
		common.TabLog:       views.NewLogView(gitSvc, styles),
		common.TabDiff:      views.NewDiffView(gitSvc, styles),
		common.TabBranches:  views.NewBranchView(gitSvc, styles),
		common.TabStash:     views.NewStashView(gitSvc, styles),
		common.TabRemotes:   views.NewRemoteView(gitSvc, styles),
		common.TabRebase:    views.NewRebaseView(gitSvc, styles),
		common.TabConflicts: views.NewConflictView(gitSvc, styles),
		common.TabWorktrees: views.NewWorktreeView(gitSvc, styles),
		common.TabBisect:    views.NewBisectView(gitSvc, styles),
	}

	model := app.New(gitSvc, cfg, viewMap)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Start filesystem watcher — only watches .git internals, safe for huge monorepos.
	if watchCh, stop, watchErr := watcher.Watch(cliSvc.RepoRoot(), cliSvc.GitDir(), 500*time.Millisecond); watchErr == nil {
		defer stop()
		go func() {
			for range watchCh {
				p.Send(common.RefreshMsg{})
			}
		}()
	}

	_, err = p.Run()
	return err
}
