package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
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
	rootCmd.AddCommand(buildZedCmd())

	rootCmd.Flags().StringP("path", "p", ".", "Path to the git repository")

	return rootCmd
}

type zedTask struct {
	Label               string            `json:"label"`
	Command             string            `json:"command"`
	Args                []string          `json:"args,omitempty"`
	Env                 map[string]string `json:"env,omitempty"`
	Cwd                 string            `json:"cwd,omitempty"`
	UseNewTerminal      bool              `json:"use_new_terminal,omitempty"`
	AllowConcurrentRuns bool              `json:"allow_concurrent_runs,omitempty"`
	Reveal              string            `json:"reveal,omitempty"`
	Hide                string            `json:"hide,omitempty"`
	Shell               string            `json:"shell,omitempty"`
	ShowSummary         bool              `json:"show_summary,omitempty"`
	ShowCommand         bool              `json:"show_command,omitempty"`
}

const zedLabelPrefix = "zgv:"

func buildZedCmd() *cobra.Command {
	zedCmd := &cobra.Command{
		Use:   "zed",
		Short: "Manage Zed IDE integration",
		Long: `Manage global Zed tasks for zgv.

Examples:
  zgv zed status
  zgv zed install
  zgv zed uninstall`,
	}

	zedCmd.AddCommand(buildZedInstallCmd())
	zedCmd.AddCommand(buildZedUninstallCmd())
	zedCmd.AddCommand(buildZedStatusCmd())

	return zedCmd
}

func buildZedInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install global Zed tasks for zgv",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfgDir, err := zedConfigDir()
			if err != nil {
				return err
			}
			tasksPath := filepath.Join(cfgDir, "tasks.json")

			existing, err := readZedTasks(tasksPath)
			if err != nil {
				return err
			}

			merged := mergeZedTasks(existing, defaultZedTasks())
			if err := writeZedTasks(tasksPath, merged); err != nil {
				return err
			}

			fmt.Printf("Installed zgv Zed integration at %s\n", tasksPath)
			fmt.Println("Open Zed and run: task: spawn -> zgv:*")
			return nil
		},
	}
}

func buildZedUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove global Zed tasks managed by zgv",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfgDir, err := zedConfigDir()
			if err != nil {
				return err
			}
			tasksPath := filepath.Join(cfgDir, "tasks.json")

			existing, err := readZedTasks(tasksPath)
			if err != nil {
				return err
			}
			cleaned := removeManagedZedTasks(existing)

			if err := writeZedTasks(tasksPath, cleaned); err != nil {
				return err
			}
			fmt.Printf("Removed zgv Zed integration from %s\n", tasksPath)
			return nil
		},
	}
}

func buildZedStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show global Zed integration status",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfgDir, err := zedConfigDir()
			if err != nil {
				return err
			}
			tasksPath := filepath.Join(cfgDir, "tasks.json")

			existing, err := readZedTasks(tasksPath)
			if err != nil {
				return err
			}

			var labels []string
			for _, t := range existing {
				if strings.HasPrefix(t.Label, zedLabelPrefix) {
					labels = append(labels, t.Label)
				}
			}

			fmt.Printf("Zed tasks file: %s\n", tasksPath)
			if len(labels) == 0 {
				fmt.Println("zgv integration: not installed")
				return nil
			}

			fmt.Printf("zgv integration: installed (%d task(s))\n", len(labels))
			for _, label := range labels {
				fmt.Printf("  - %s\n", label)
			}
			return nil
		},
	}
}

func zedConfigDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("ZGV_ZED_CONFIG_DIR")); override != "" {
		return override, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}

	xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if xdg != "" {
		return filepath.Join(xdg, "zed"), nil
	}

	return filepath.Join(home, ".config", "zed"), nil
}

func readZedTasks(path string) ([]zedTask, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read zed tasks file %s: %w", path, err)
	}

	var tasks []zedTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse zed tasks file %s: %w", path, err)
	}
	return tasks, nil
}

func writeZedTasks(path string, tasks []zedTask) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create zed config dir: %w", err)
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize zed tasks: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write zed tasks file %s: %w", path, err)
	}
	return nil
}

func mergeZedTasks(existing, managed []zedTask) []zedTask {
	cleaned := removeManagedZedTasks(existing)
	return append(cleaned, managed...)
}

func removeManagedZedTasks(tasks []zedTask) []zedTask {
	out := make([]zedTask, 0, len(tasks))
	for _, t := range tasks {
		if strings.HasPrefix(t.Label, zedLabelPrefix) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func defaultZedTasks() []zedTask {
	return []zedTask{
		{
			Label:               "zgv: open (current worktree)",
			Command:             "zgv",
			Args:                []string{"--path", "$ZED_WORKTREE_ROOT"},
			Cwd:                 "$ZED_WORKTREE_ROOT",
			UseNewTerminal:      true,
			AllowConcurrentRuns: false,
			Reveal:              "always",
			Hide:                "never",
			Shell:               "system",
			ShowSummary:         true,
			ShowCommand:         true,
		},
		{
			Label:               "zgv: dev hot reload",
			Command:             "task",
			Args:                []string{"dev"},
			Cwd:                 "$ZED_WORKTREE_ROOT",
			UseNewTerminal:      true,
			AllowConcurrentRuns: false,
			Reveal:              "always",
			Hide:                "never",
			Shell:               "system",
			ShowSummary:         true,
			ShowCommand:         true,
		},
		{
			Label:               "zgv: check (fmt+vet+lint+test)",
			Command:             "task",
			Args:                []string{"check"},
			Cwd:                 "$ZED_WORKTREE_ROOT",
			UseNewTerminal:      true,
			AllowConcurrentRuns: false,
			Reveal:              "always",
			Hide:                "never",
			Shell:               "system",
			ShowSummary:         true,
			ShowCommand:         true,
		},
		{
			Label:               "zgv: release patch",
			Command:             "task",
			Args:                []string{"release", "--", "patch"},
			Cwd:                 "$ZED_WORKTREE_ROOT",
			UseNewTerminal:      true,
			AllowConcurrentRuns: false,
			Reveal:              "always",
			Hide:                "never",
			Shell:               "system",
			ShowSummary:         true,
			ShowCommand:         true,
		},
		{
			Label:               "zgv: release minor",
			Command:             "task",
			Args:                []string{"release", "--", "minor"},
			Cwd:                 "$ZED_WORKTREE_ROOT",
			UseNewTerminal:      true,
			AllowConcurrentRuns: false,
			Reveal:              "always",
			Hide:                "never",
			Shell:               "system",
			ShowSummary:         true,
			ShowCommand:         true,
		},
		{
			Label:               "zgv: release major",
			Command:             "task",
			Args:                []string{"release", "--", "major"},
			Cwd:                 "$ZED_WORKTREE_ROOT",
			UseNewTerminal:      true,
			AllowConcurrentRuns: false,
			Reveal:              "always",
			Hide:                "never",
			Shell:               "system",
			ShowSummary:         true,
			ShowCommand:         true,
		},
	}
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
