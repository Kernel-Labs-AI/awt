# AWT - Agent WorkTrees

**AWT** (Agent WorkTrees) is a CLI tool that enables multiple AI agents to safely create, use, and hand off Git worktrees for concurrent development workflows.

[![CI](https://github.com/kernel-labs-ai/awt/workflows/CI/badge.svg)](https://github.com/kernel-labs-ai/awt/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/kernel-labs-ai/awt)](https://goreportcard.com/report/github.com/kernel-labs-ai/awt)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Overview

AWT solves the problem of multiple AI agents (like Claude, GPT, etc.) working on the same Git repository simultaneously. It provides:

- **Isolated Worktrees**: Each agent gets its own working directory
- **Safe Concurrency**: File-based locking prevents conflicts
- **Task Lifecycle**: Track tasks from creation through handoff
- **Branch Management**: Automatic branch creation and cleanup
- **Metadata Tracking**: Persistent task state and history

## Features

- 🔒 **Safe Concurrency**: POSIX file locking with timeout/retry
- 🌳 **Git Worktrees**: Isolated working directories per task
- 📋 **Task Tracking**: JSON-based metadata with atomic writes
- 🔄 **Lifecycle Management**: NEW → ACTIVE → HANDOFF_READY → MERGED
- 🛡️ **Safety Rails**: Input validation, path checking, CWD safety
- ⚙️ **Configuration**: Multi-level config (system/user/repo/env)
- 📊 **Logging**: Structured logging with levels and fields
- 🚀 **Fast**: Written in Go, minimal dependencies

## Installation

### macOS (Homebrew)

```bash
brew install kernel-labs-ai/tap/awt
```

### Linux/macOS (Manual)

```bash
# Download the latest release
curl -LO https://github.com/kernel-labs-ai/awt/releases/latest/download/awt_<VERSION>_<OS>_<ARCH>.tar.gz

# Extract and install
tar xzf awt_*.tar.gz
sudo mv awt /usr/local/bin/

# Verify installation
awt version
```

### From Source

```bash
git clone https://github.com/kernel-labs-ai/awt.git
cd awt
go build -o awt ./cmd/awt
sudo mv awt /usr/local/bin/
```

## Quick Start

```bash
# Initialize AWT in your repository
cd /path/to/your/repo
awt init

# Start a new task
awt task start --agent=claude --title="Add user authentication"
# Output: Task ID, branch name, worktree path

# Work in the worktree
cd .awt/wt/<task-id>
# Make changes...

# Commit changes
awt task commit <task-id> -m "Implement login endpoint"

# Sync with base branch
awt task sync <task-id>

# Complete and hand off (push + create PR)
awt task handoff <task-id>

# List all tasks
awt list

# Clean up orphaned tasks
awt prune
```

## Commands

### Core Commands

#### `awt init`
Initialize AWT in a Git repository. Creates necessary directories and version file.

```bash
awt init [--repo=<path>]
```

#### `awt task start`
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

#### `awt task status`
Show task status and metadata.

```bash
awt task status [task-id] [--branch=<name>] [--json]
```

#### `awt task commit`
Commit changes in a task's worktree.

```bash
awt task commit [task-id] -m "<message>" [options]

Options:
  -m, --message string   Commit message
  -a, --all             Stage all modified files
  --signoff             Add Signed-off-by trailer
  --gpg-sign string     GPG sign commit
```

#### `awt task sync`
Sync task branch with base branch.

```bash
awt task sync [task-id] [options]

Options:
  --merge              Use merge instead of rebase
  --no-fetch           Skip fetching remote
```

#### `awt task handoff`
Complete task and hand off (push + create PR + detach worktree).

```bash
awt task handoff [task-id] [options]

Options:
  --no-push            Don't push to remote
  --no-pr              Don't create pull request
  --keep-worktree      Keep worktree after handoff
  --force-remove       Remove worktree even if CWD is inside it
```

### Additional Commands

#### `awt task exec`
Execute a command in task's worktree.

```bash
awt task exec <task-id> -- <command> [args...]
```

#### `awt task checkout`
Checkout existing task for review.

```bash
awt task checkout <task-id> [--path=<path>]
```

#### `awt task adopt`
Adopt an existing Git branch as AWT task.

```bash
awt task adopt <branch> --agent=<name> [--title="<title>"]
```

#### `awt task unlock`
Unlock a task branch by detaching worktrees.

```bash
awt task unlock <task-id> [--remove]
```

#### `awt list`
List all tasks with status.

```bash
awt list [--json]
```

#### `awt prune`
Clean up orphaned tasks and stale locks.

```bash
awt prune [--dry-run] [--json]
```

### Configuration

#### `awt config list`
Show all configuration settings.

```bash
awt config list [--json]
```

#### `awt config get`
Get a configuration value.

```bash
awt config get <key>
```

#### `awt config set`
Set a configuration value.

```bash
awt config set <key> <value> [--scope=user|repo|system]
```

#### `awt config unset`
Unset a configuration value.

```bash
awt config unset <key> [--scope=user|repo|system]
```

#### `awt config path`
Show configuration file path.

```bash
awt config path [--scope=user|repo|system]
```

## Configuration

AWT supports multi-level configuration with the following precedence (highest to lowest):

1. Environment variables (highest)
2. Repository config (`.git/awt/config.json`)
3. User config (`~/.config/awt/config.json`)
4. System config (`/etc/awt/config.json`)

### Available Settings

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

### Example Configuration

```json
{
  "default_agent": "claude",
  "branch_prefix": "agent",
  "worktree_dir": "./worktrees",
  "rebase_default": true,
  "auto_push": true,
  "auto_pr": true,
  "remote_name": "origin",
  "lock_timeout": 60,
  "verbose_git": false
}
```

## Architecture

### Task States

```
NEW → ACTIVE → HANDOFF_READY → MERGED
  ↓      ↓           ↓
  └──→ ABANDONED ←───┘
```

- **NEW**: Task created but not yet active
- **ACTIVE**: Work in progress
- **HANDOFF_READY**: PR created, ready for review
- **MERGED**: PR merged, task complete
- **ABANDONED**: Task cancelled or abandoned

### Directory Structure

```
your-repo/
├── .git/
│   └── awt/
│       ├── version          # AWT version
│       ├── config.json      # Repository config
│       ├── tasks/           # Task metadata
│       │   └── <id>.json
│       └── locks/           # Lock files
│           ├── global.lock
│           └── task-<id>.lock
└── .awt/
    └── wt/                  # Worktrees
        └── <id>/            # Task worktree
```

### Task Metadata

Each task is stored as JSON at `.git/awt/tasks/<id>.json`:

```json
{
  "id": "20251110-120000-abc123",
  "agent": "claude",
  "title": "Add user authentication",
  "branch": "awt/claude/20251110-120000-abc123",
  "base": "origin/main",
  "created_at": "2025-11-10T12:00:00Z",
  "state": "ACTIVE",
  "worktree_path": "/path/to/repo/.awt/wt/20251110-120000-abc123",
  "last_commit": "sha1...",
  "pr_url": ""
}
```

## Safety Features

### Input Validation

- **Agent Names**: Alphanumeric, dash, underscore only (max 50 chars)
- **Task Titles**: No newlines/tabs, max 200 chars
- **Branch Names**: Git naming rules enforced
- **Commit Messages**: Subject line max 100 chars, total max 10000 chars

### Path Safety

- Worktree path validation (existence, emptiness)
- CWD checks before worktree removal
- No operations inside `.git` directory
- Worktree != repository root

### Concurrency Safety

- POSIX file locking (flock) with EWOULDBLOCK/EAGAIN checks
- O_EXCL fallback for network filesystems
- Global lock for repository-wide operations
- Per-task locks for task-specific operations
- Configurable timeouts and retry logic

## Use Cases

### Multiple AI Agents Working Simultaneously

```bash
# Agent 1 (Claude)
awt task start --agent=claude --title="Add user auth"

# Agent 2 (GPT)
awt task start --agent=gpt --title="Add API docs"

# Both agents work independently in their worktrees
# No conflicts, no stepping on each other's toes
```

### Code Review Workflow

```bash
# Reviewer checks out the task
awt task checkout 20251110-120000-abc123

# Review code in separate worktree
cd .awt/wt/20251110-120000-abc123
# Make review comments, test changes...

# Return to main worktree when done
cd ../../..
```

### Task Handoff Between Agents

```bash
# Agent 1 completes their work
awt task handoff 20251110-120000-abc123

# Agent 2 takes over for review
awt task checkout 20251110-120000-abc123

# Agent 2 makes changes and re-commits
awt task commit 20251110-120000-abc123 -m "Address review comments"
awt task handoff 20251110-120000-abc123
```

## Troubleshooting

### Lock Timeout

```bash
# Error: Lock timeout
# Solution: Increase timeout or check for stale locks
awt config set lock_timeout 60
awt prune  # Clean up stale locks
```

### Branch Already Exists

```bash
# Error: Branch already exists
# Solution: Use different task ID or delete old branch
git branch -d awt/agent/old-task-id
```

### Worktree Not Found

```bash
# Error: Worktree not found
# Solution: Check task status and recreate if needed
awt task status <task-id>
awt prune  # Clean up orphaned metadata
```

### CWD Inside Worktree

```bash
# Error: Cannot remove worktree, CWD is inside it
# Solution: Change directory or use --force-remove
cd /path/to/main/worktree
awt task handoff <task-id>
# Or
awt task handoff <task-id> --force-remove
```

## Development

### Building

```bash
go build -o awt ./cmd/awt
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Releasing

```bash
# Tag a new version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub Actions will automatically:
# - Run tests
# - Build binaries for all platforms
# - Create GitHub release
# - Update Homebrew tap
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `go test ./...` and `go vet ./...`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Credits

Developed by [Kernel Labs](https://kernellabs.ai) for enabling AI agent collaboration.

## Support

- **Issues**: [GitHub Issues](https://github.com/kernel-labs-ai/awt/issues)
- **Discussions**: [GitHub Discussions](https://github.com/kernel-labs-ai/awt/discussions)
- **Documentation**: [Wiki](https://github.com/kernel-labs-ai/awt/wiki)
