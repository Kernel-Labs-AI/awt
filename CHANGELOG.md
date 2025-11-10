# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of AWT (Agent WorkTrees)
- Core task lifecycle management (NEW → ACTIVE → HANDOFF_READY → MERGED)
- Git worktree isolation for concurrent agent work
- POSIX file locking for safe concurrency
- JSON-based task metadata with atomic writes
- Multi-level configuration system (system/user/repo/env)
- Comprehensive safety validation
- Structured logging with levels and fields
- Commands:
  - `awt init` - Initialize AWT in repository
  - `awt task start` - Start new task with worktree
  - `awt task status` - Show task status
  - `awt task exec` - Execute command in worktree
  - `awt task commit` - Commit changes
  - `awt task sync` - Sync with base branch
  - `awt task handoff` - Complete and hand off task
  - `awt task checkout` - Checkout existing task
  - `awt task adopt` - Adopt existing branch
  - `awt task unlock` - Unlock task branch
  - `awt list` - List all tasks
  - `awt prune` - Clean up orphaned tasks
  - `awt config` - Manage configuration
- Cross-platform support (Linux, macOS, Windows)
- Homebrew tap for easy installation
- GitHub Actions CI/CD pipeline
- Comprehensive documentation

### Security
- Input validation for all user inputs
- Path safety checks
- CWD safety before worktree removal
- Git command execution via -C flag
- No shell injection vulnerabilities

## [0.1.0] - 2025-11-10

### Added
- Project scaffolding
- Basic Git worktree operations
- Task metadata structure
- Lock mechanism implementation

[Unreleased]: https://github.com/decibelvc/awt/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/decibelvc/awt/releases/tag/v0.1.0
