# Zed project tasks

This repository ships workspace tasks in `.zed/tasks.json` so `zgv` can be run
from Zed without manual terminal setup.

## Use

1. Open this repository as a project in Zed.
2. Open the command palette and run `task: spawn`.
3. Start with one of:
   - `zgv: open (current worktree)`
   - `zgv: dev hot reload`
   - `zgv: check (fmt+vet+lint+test)`
   - `zgv: release patch` / `minor` / `major`

## Optional keybindings

You can bind a key directly to a task in your Zed keymap:

```json
{
  "context": "Workspace",
  "bindings": {
    "alt-g": ["task::Spawn", { "task_name": "zgv: open (current worktree)" }]
  }
}
```

See Zed task docs: https://zed.dev/docs/tasks
