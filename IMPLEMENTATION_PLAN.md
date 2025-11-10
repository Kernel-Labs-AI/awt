# AWT Implementation Plan

Version: v0.1
Last Updated: 2025-11-10

## Overview

This document outlines the implementation plan for AWT (Agent WorkTrees), a CLI tool that enables multiple AI agents to safely create, use, and hand off Git worktrees. The project is structured into 8 phases with 29 distinct tasks, each tracked in Linear.

## Project Structure

```
awt/
├── cmd/
│   └── awt/           # Main CLI entry point
├── internal/
│   ├── config/        # Configuration management
│   ├── errors/        # Error types and exit codes
│   ├── git/           # Git operations wrapper
│   ├── lock/          # POSIX locking mechanism
│   ├── repo/          # Repository discovery
│   └── task/          # Task data model and persistence
├── test/
│   ├── integration/   # Integration tests
│   └── testdata/      # Test fixtures
├── .goreleaser.yml    # Release configuration
├── go.mod
└── README.md
```

## Implementation Phases

### Phase 1: Foundation (6 tasks)

Core infrastructure that all other components depend on.

#### DEC-153: Setup Go project structure and dependencies
- Create `go.mod` with Go 1.21+
- Setup directory structure (`cmd/awt`, `internal/`)
- Add core dependencies:
  - `github.com/spf13/cobra` for CLI
  - `golang.org/x/sys/unix` for flock
- Create basic `main.go` with version command
- Setup `.gitignore` for Go projects

#### DEC-154: Implement repository discovery and path resolution
- Implement `git rev-parse --show-toplevel` to find `WT_ROOT`
- Implement `git rev-parse --git-common-dir` to find `GIT_COMMON`
- Support `--repo` flag to override discovery
- Add validation for Git version >= 2.33
- Create package `internal/repo` with `DiscoverRepo()` function

#### DEC-155: Design and implement Task data model
- Define Task struct with all fields:
  - `id`, `agent`, `title`, `branch`, `base`
  - `created_at`, `state`, `worktree_path`
  - `last_commit`, `pr_url`
- Implement JSON serialization/deserialization
- Create atomic file write helper (write temp + rename)
- Implement task persistence at `$GIT_COMMON/awt/tasks/<id>.json`
- Add task loading and validation logic
- Create package `internal/task`

#### DEC-156: Implement POSIX locking mechanism
- Implement flock-based locking using `golang.org/x/sys/unix`
- Add fallback to `O_CREAT|O_EXCL` for network filesystems
- Create global lock at `$GIT_COMMON/awt/locks/global.lock`
- Create per-task locks at `$GIT_COMMON/awt/locks/<id>.lock`
- Add timeout and retry logic
- Handle lock cleanup on SIGINT/SIGTERM
- Create package `internal/lock`

#### DEC-169: Create Git primitives wrapper package
Implement Git operations wrapper (`internal/git`):
- Always execute with `-C WT_ROOT`
- Wrapper for: worktree add/remove/list, fetch, rebase, merge, switch
- Capture and parse stdout/stderr
- Propagate exit codes
- Handle `--verbose` flag to log commands
- Parse `git worktree list --porcelain`
- Branch existence and checkout checks

#### DEC-170: Implement error handling and exit codes
- Define all error codes (10-61 as per spec):
  - `10` REPO_NOT_FOUND
  - `11` GIT_TOO_OLD
  - `20` BRANCH_EXISTS
  - `21` BRANCH_CHECKED_OUT_ELSEWHERE
  - `22` WORKTREE_EXISTS
  - `23` DETACH_FAILED
  - `24` REMOVE_FAILED
  - `30` SYNC_CONFLICTS
  - `31` PUSH_REJECTED
  - `40` LOCK_TIMEOUT
  - `41` LOCK_HELD
  - `50` TOOL_MISSING
  - `60` INVALID_TASK_ID
  - `61` CASE_ONLY_COLLISION
- Create custom error types for each category
- Implement error formatting (single-line + JSON)
- Add `--json` error output with code and hint
- Create package `internal/errors`

### Phase 2: Core Commands (4 tasks)

Basic command implementations for task lifecycle.

#### DEC-157: Implement `awt init` command
- Verify inside Git repository
- Check Git version >= 2.33
- Create directory structure: `$GIT_COMMON/awt/{tasks,locks}`
- Write version file
- Handle already-initialized state gracefully
- Add command to cobra CLI

#### DEC-158: Implement `awt task start` command
- Parse flags: `--base`, `--agent`, `--title`, `--id`, `--no-fetch`
- Generate unique task ID: `YYYYmmdd-HHMMSS-<6random>`
- Create branch name: `awt/<agent>/<id>`
- Fetch unless `--no-fetch`
- Validate branch doesn't exist and worktree path is available
- Run `git worktree add -b <branch> <worktree_path> <base>`
- Create and save task JSON with `state=ACTIVE`
- Update ref: `refs/awt/tasks/<id>`
- Output JSON with `id`, `branch`, `worktree_path`

#### DEC-159: Implement `awt task status` command
- Accept task id, `--branch`, or infer from current worktree
- Load task metadata from JSON
- Display: state, branch, base, last_commit, pr_url
- Support `--json` output format
- Handle `INVALID_TASK_ID` error gracefully

#### DEC-160: Implement `awt task exec` command
- Accept task id and command args after `--`
- Load task and verify worktree exists
- Execute command inside worktree with proper working directory
- Stream stdout/stderr to parent
- Return child process exit code
- Handle signal propagation (SIGINT, SIGTERM)

### Phase 3: Task Lifecycle (3 tasks)

Complete the agent workflow from commit to handoff.

#### DEC-161: Implement `awt task commit` command
- Accept task id and flags: `--message`, `--all`, `--signoff`, `--gpg-sign`
- Stage files: respect index or use `--all`
- Generate default message if not provided:
  ```
  feat(task:<id>): <title>

  <metadata body>
  ```
- Execute git commit in worktree
- Update `task.last_commit` with new SHA
- Save updated task metadata

#### DEC-162: Implement `awt task sync` command
- Accept task id and flags: `--rebase`, `--merge`, `--submodules`
- Fetch base ref
- Handle shallow clones (unshallow if needed)
- Default to rebase, use merge if `--merge` specified
- Execute git rebase/merge in worktree
- On conflicts: exit with `SYNC_CONFLICTS` code
- Update submodules if `--submodules` flag set

#### DEC-163: Implement `awt task handoff` command
The most complex command - coordinates multiple steps:
- Accept flags: `--push`, `--create-pr`, `--keep-worktree`, `--force-remove`
- **Step 1**: Run commit (allow no-op)
- **Step 2**: Run sync (propagate conflicts)
- **Step 3**: Push if `--push`: `git push -u origin <branch>`
- **Step 4**: Detach HEAD in worktree: `git switch --detach`
- **Step 5**: Create PR/MR if `--create-pr` (check gh/glab availability)
- **Step 6**: Remove worktree with safety checks:
  - Skip removal if CWD is inside and no `--force-remove`
  - If `--force-remove` and CWD inside: chdir to `WT_ROOT` then remove
- Update state to `HANDOFF_READY`
- Handle idempotency (rerunnable after crashes)

### Phase 4: Task Management (5 tasks)

Additional utilities for managing tasks and worktrees.

#### DEC-164: Implement `awt task checkout` command
Create task checkout for human validation:
- Accept task id or `--branch`
- Parse `--path` (default: `./wt/<id>`) and `--submodules`
- Create new worktree at specified path
- Checkout task branch
- Initialize/update submodules if `--submodules`
- Output worktree path

#### DEC-165: Implement `awt task adopt` command
- Accept `--branch` (required) and optional `--id`, `--agent`, `--base`, `--title`
- Verify branch exists
- Detect base via merge-base if not provided
- Generate id if not provided
- Create task metadata with `state=ACTIVE`
- Set `worktree_path` empty until checkout
- Save task JSON

#### DEC-166: Implement `awt task unlock` command
- Accept task id or `--branch`
- Find worktrees where branch is checked out
- Force detach HEAD in those worktrees
- Optionally remove worktree if empty and safe
- Free up the branch for other operations

#### DEC-167: Implement `awt list` command
- Enumerate all tasks from `$GIT_COMMON/awt/tasks/*.json`
- Enrich with current checkout status from `git worktree list`
- Check remote branch presence
- Display table with: id, agent, title, state, branch
- Support `--json` output format

#### DEC-168: Implement `awt prune` command
- Run `git worktree prune`
- Find orphaned task metadata (tasks with non-existent worktrees)
- Clean up orphaned `tasks/*.json` files
- Remove stale locks
- Make operation idempotent and safe
- Output list of pruned resources

### Phase 5: Features & Safety (3 tasks)

Production-ready features for configuration and safety.

#### DEC-171: Implement configuration management
Create config system (`internal/config`):
- Define all environment variables:
  - `AWT_DEFAULT_BASE` (default: `origin/main`)
  - `AWT_BRANCH_PREFIX` (default: `awt`)
  - `AWT_WORKTREE_DIR` (default: `.awt/wt`)
  - `AWT_PR_TOOL` (`gh|glab|none`)
  - `AWT_NO_FETCH=1` (disables fetch by default)
  - `AWT_LOG=1` (enables file logging)
- Implement config file at `$GIT_COMMON/awt/config.json`
- Establish precedence: **CLI flag > env > config > default**
- Add getters for all config values
- Support config validation

#### DEC-172: Implement safety rails and validation
- Destructive ops only for `awt/` prefixed branches unless `--force`
- Prevent worktree removal if CWD is inside it (unless `--force-remove`)
- Sanitize agent, title, id (reject shell metacharacters)
- macOS case-folding guard (reject case-only-different branches)
- Add `CASE_ONLY_COLLISION` error code
- Submodule and LFS detection with hints

#### DEC-173: Implement observability and logging
- Implement `--verbose` flag to print Git commands and durations
- Optional log file at `$GIT_COMMON/awt/awt.log` when `AWT_LOG=1`
- Structured logging (timestamp, level, message)
- Signal handling (SIGINT/SIGTERM) with cleanup
- No telemetry (privacy-first)

### Phase 6: Testing (4 tasks)

Comprehensive test coverage including unit, integration, and fuzz tests.

#### DEC-174: Write unit tests for core components
Target: **>80% code coverage** for internal packages

1. **Path discovery** from nested directories
2. **Metadata I/O** with atomic writes and corrupted JSON handling
3. **Locking** with contention, serialization, and timeouts
4. **Name generation**: ID uniqueness, branch sanitization, macOS case
5. **Git wrapper**: command execution, output capture, exit codes

#### DEC-175: Write integration tests - happy paths
Create hermetic test harness:
- Temp repo with bare origin
- Helper functions: `gitInit()`, `seedCommits()`, `runAWT()`

Tests:
- **Test 6**: Start → Commit → Sync → Handoff → Checkout full flow
- **Test 7**: Multiple concurrent agents (3 tasks) with isolation
- **Test 8**: Start from any directory (root, nested, inside worktree)
- **Test 9**: Adopt existing branch flow
- **Test 10**: PR creation with gh/glab

#### DEC-176: Write integration tests - edge cases
- **Test 11**: Branch already checked out - handoff detach behavior
- **Test 12**: Rebase conflict - SYNC_CONFLICTS and recovery
- **Test 13**: Push rejected - PUSH_REJECTED handling
- **Test 14**: Shallow clone - auto-unshallow
- **Test 15**: Submodules with `--submodules`
- **Test 16**: Crash during handoff - idempotency
- **Test 17**: Unlock stuck branch
- **Test 18**: Force remove safety with CWD checks
- **Test 19**: Large files and LFS warnings
- **Test 20**: Config precedence validation

#### DEC-177: Write property, fuzz, and CLI tests
**Property/Fuzz tests:**
- **Test 23**: Branch/name fuzzing - randomized agent names and titles
- **Test 24**: Concurrency stress - 10 tasks with interleaved operations

**CLI UX tests:**
- **Test 21**: JSON output contract validation
- **Test 22**: Prune idempotency
- **Test 25**: Help text completeness
- **Test 26**: Version output format

### Phase 7: Distribution (3 tasks)

Cross-platform builds and package distribution.

#### DEC-178: Setup GoReleaser for cross-platform builds
- Create `.goreleaser.yml` configuration
- Target platforms:
  - `darwin/amd64`
  - `darwin/arm64`
  - `linux/amd64`
  - `linux/arm64`
- Generate static binaries
- Create tar.gz archives
- Include README, LICENSE
- Setup version injection from Git tags
- Test local builds with `goreleaser build --snapshot`

#### DEC-179: Setup GitHub Actions CI/CD pipeline
- **Test workflow**: run tests on `macos-latest` and `ubuntu-latest`
- Matrix build for Go versions
- Run unit and integration tests
- Code coverage reporting
- **Release workflow**: trigger GoReleaser on tags
- Security: dependabot configuration

#### DEC-180: Create Homebrew tap and formula
- Create `homebrew-tap` repository
- Generate formula for awt
- Configure for both macOS and Linuxbrew
- Add `brew test` command that runs `awt version`
- Setup automatic formula updates from GoReleaser
- Document installation: `brew install <org>/tap/awt`

### Phase 8: Documentation (1 task)

#### DEC-181: Write comprehensive documentation
- **README.md**: overview, installation, quick start
- **CONTRIBUTING.md**: development setup
- Command reference documentation
- Architecture documentation
- Examples for common workflows
- Troubleshooting guide
- FAQ for edge cases
- PR template and issue templates

## Recommended Implementation Order

1. **Phase 1 (Foundation)** - Start here, these are dependencies for everything
   - Begin with DEC-153, DEC-154, DEC-155
   - Then DEC-156, DEC-169, DEC-170

2. **Phase 2 (Core Commands)** - Get basic functionality working
   - DEC-157 (init) first
   - Then DEC-158 (start), DEC-159 (status), DEC-160 (exec)

3. **Phase 3 (Task Lifecycle)** - Complete the agent workflow
   - DEC-161 (commit), DEC-162 (sync)
   - DEC-163 (handoff) last - it's the most complex

4. **Phase 4 (Task Management)** - Add utility commands
   - Can be done in any order based on priority

5. **Phase 5 (Features & Safety)** - Production readiness
   - DEC-171 (config) early for flexibility
   - DEC-172 (safety) before release
   - DEC-173 (logging) helps with debugging

6. **Phase 6 (Testing)** - Write tests throughout development
   - Start unit tests (DEC-174) alongside implementation
   - Integration tests (DEC-175, DEC-176) after commands work
   - Fuzz tests (DEC-177) before v0.1 release

7. **Phase 7 (Distribution)** - Setup once core is stable
   - DEC-178 (GoReleaser) first
   - DEC-179 (CI/CD) to automate
   - DEC-180 (Homebrew) for easy installation

8. **Phase 8 (Documentation)** - Finalize before release
   - DEC-181 before announcing v0.1

## State Machine

```
NEW → ACTIVE → HANDOFF_READY → MERGED | ABANDONED
```

- **NEW**: Metadata reserved (fast pass-through)
- **ACTIVE**: Worktree exists and owns the branch
- **HANDOFF_READY**: Changes committed/pushed, branch detached
- **MERGED**: Integrated into base branch
- **ABANDONED**: Closed without merge

## Key Design Decisions

### Metadata Storage
- **Location**: `$GIT_COMMON/awt/`
- **Format**: JSON for tasks, refs for base anchoring
- **Atomicity**: Write to temp file + rename

### Concurrency
- **Global lock**: Serializes worktree creation/removal
- **Per-task lock**: Serializes state transitions
- **Mechanism**: POSIX flock with O_CREAT|O_EXCL fallback

### Naming Conventions
- **Task ID**: `YYYYmmdd-HHMMSS-<6random>`
- **Branch**: `awt/<agent>/<id>`
- **Worktree Path**: `<WT_ROOT>/.awt/wt/<id>`

### Safety Features
- Branch prefix restrictions
- CWD checks before removal
- Input sanitization
- macOS case-folding guards

## Acceptance Criteria

- [ ] All tests pass on macOS and Linux CI
- [ ] Handoff always leaves task branches free in the repo
- [ ] Two or more agents can run start→handoff concurrently without collisions
- [ ] Homebrew install works and `brew test awt` runs `awt version`
- [ ] Case-only branch collisions prevented on macOS
- [ ] POSIX locks work; fallback atomic locking works on network FS

## Linear Project

All tasks are tracked in Linear:
https://linear.app/decibelvc/project/awt-agents-worktrees-0a719bfe86dc

Task IDs: **DEC-153** through **DEC-181** (29 tasks total)

## References

- Full specification: See project description in Linear
- Git worktrees documentation: https://git-scm.com/docs/git-worktree
- POSIX flock: `man 2 flock`
