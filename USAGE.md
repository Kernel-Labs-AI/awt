# AWT Command Reference

AWT (Agent WorkTrees) enables multiple AI agents to safely create and use Git worktrees for concurrent development.

## Quick Start

```bash
awt init                                              # Initialize AWT in your repo
awt task start --agent=claude --title="Your task"    # Start a new task
cd .awt/wt/<task-id>                                  # Work in the worktree
awt task commit <task-id> -m "Your message"          # Commit changes
awt task sync <task-id>                               # Sync with base branch
awt task handoff <task-id>                            # Push + create PR
awt list                                              # List all tasks
awt prune                                             # Clean up orphaned tasks
```

## Commands

### `awt init`
Initialize AWT in a Git repository.
```bash
awt init [--repo=<path>]
```

### `awt task start`
Start a new task with isolated worktree.
```bash
awt task start --agent=<name> --title="<description>" [options]

Options:
  --agent string       Agent name (required)
  --title string       Task title (required)
  --base string        Base branch (default: origin/main)
  --id string          Custom task ID (auto-generated if not provided)
  --no-fetch           Skip git fetch
  --json               Output as JSON
```

### `awt task status`
Show task status and metadata.
```bash
awt task status [task-id] [--branch=<name>] [--json]
```

### `awt task commit`
Commit changes in a task's worktree.
```bash
awt task commit [task-id] -m "<message>" [options]

Options:
  -m, --message string   Commit message
  -a, --all             Stage all modified files
  --signoff             Add Signed-off-by trailer
  --gpg-sign string     GPG sign commit
```

### `awt task sync`
Sync task branch with base branch.
```bash
awt task sync [task-id] [options]

Options:
  --merge              Use merge instead of rebase
  --no-fetch           Skip fetching remote
```

### `awt task handoff`
Complete task and hand off (push + create PR + detach worktree).
```bash
awt task handoff [task-id] [options]

Options:
  --no-push            Don't push to remote
  --no-pr              Don't create pull request
  --keep-worktree      Keep worktree after handoff
  --force-remove       Remove worktree even if CWD is inside it
```

### `awt task exec`
Execute a command in task's worktree.
```bash
awt task exec <task-id> -- <command> [args...]
```

### `awt task editor`
Open your default editor in task's worktree.
```bash
awt task editor [task-id] [options]

Options:
  --editor string   Editor to use (defaults to $EDITOR)
  --branch string   Branch name
  --repo string     Path to Git repository
```

### `awt task checkout`
Checkout existing task for review.
```bash
awt task checkout <task-id> [--path=<path>]
```

### `awt task adopt`
Adopt an existing Git branch as AWT task.
```bash
awt task adopt <branch> --agent=<name> [--title="<title>"]
```

### `awt task unlock`
Unlock a task branch by detaching worktrees.
```bash
awt task unlock <task-id> [--remove]
```

### `awt list`
List all tasks with status.
```bash
awt list [--json]
```

### `awt prune`
Clean up orphaned tasks and stale locks.
```bash
awt prune [--dry-run] [--json]
```

### `awt config list`
Show all configuration settings.
```bash
awt config list [--json]
```

### `awt config get`
Get a configuration value.
```bash
awt config get <key>
```

### `awt config set`
Set a configuration value.
```bash
awt config set <key> <value> [--scope=user|repo|system]
```

### `awt config unset`
Unset a configuration value.
```bash
awt config unset <key> [--scope=user|repo|system]
```

### `awt config path`
Show configuration file path.
```bash
awt config path [--scope=user|repo|system]
```

## Configuration Settings

| Setting | Description | Default | Env Variable |
|---------|-------------|---------|--------------|
| `default_agent` | Default agent name | `unknown` | `AWT_DEFAULT_AGENT` |
| `branch_prefix` | Branch prefix | `awt` | `AWT_BRANCH_PREFIX` |
| `worktree_dir` | Worktree directory | `./wt` | `AWT_WORKTREE_DIR` |
| `rebase_default` | Use rebase for sync | `true` | `AWT_REBASE_DEFAULT` |
| `auto_push` | Auto-push on handoff | `true` | `AWT_AUTO_PUSH` |
| `auto_pr` | Auto-create PR on handoff | `true` | `AWT_AUTO_PR` |
| `remote_name` | Default remote | `origin` | `AWT_REMOTE_NAME` |
| `lock_timeout` | Lock timeout (seconds) | `30` | `AWT_LOCK_TIMEOUT` |
| `verbose_git` | Verbose git output | `false` | `AWT_VERBOSE_GIT` |

Configuration precedence (highest to lowest):
1. Environment variables
2. Repository config (`.git/awt/config.json`)
3. User config (`~/.config/awt/config.json`)
4. System config (`/etc/awt/config.json`)

## Task States

```
NEW → ACTIVE → HANDOFF_READY → MERGED
  ↓      ↓           ↓
  └──→ ABANDONED ←───┘
```

## Directory Structure

```
your-repo/
├── .git/awt/
│   ├── version          # AWT version
│   ├── config.json      # Repository config
│   ├── tasks/           # Task metadata
│   └── locks/           # Lock files
└── .awt/wt/             # Worktrees
    └── <task-id>/       # Task worktree
```

## Common Workflows

### Multiple Agents Working Simultaneously
```bash
# Agent 1
awt task start --agent=claude --title="Add user auth"

# Agent 2
awt task start --agent=gpt --title="Add API docs"
```

### Code Review
```bash
awt task checkout <task-id>
cd .awt/wt/<task-id>
# Review code, test changes...
```

### Task Handoff
```bash
# Agent 1 completes work
awt task handoff <task-id>

# Agent 2 takes over
awt task checkout <task-id>
awt task commit <task-id> -m "Address review comments"
awt task handoff <task-id>
```
