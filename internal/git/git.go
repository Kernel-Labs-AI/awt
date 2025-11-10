package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/decibelvc/awt/internal/logger"
)

// Git represents a Git operations wrapper
type Git struct {
	// workTreeRoot is the root of the worktree
	workTreeRoot string
	// verbose enables command logging
	verbose bool
}

// New creates a new Git wrapper
func New(workTreeRoot string, verbose bool) *Git {
	return &Git{
		workTreeRoot: workTreeRoot,
		verbose:      verbose,
	}
}

// Result represents the result of a Git command execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// run executes a git command with -C workTreeRoot
func (g *Git) run(args ...string) (*Result, error) {
	// Prepend -C workTreeRoot to run from the worktree root
	fullArgs := append([]string{"-C", g.workTreeRoot}, args...)

	if g.verbose {
		logger.Debug("git %s", strings.Join(fullArgs, " "))
	}

	cmd := exec.Command("git", fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, fmt.Errorf("failed to execute git command: %w", err)
		}
	}

	return result, nil
}

// WorktreeAdd creates a new worktree with a new branch
func (g *Git) WorktreeAdd(path, branch, baseBranch string) (*Result, error) {
	return g.run("worktree", "add", "-b", branch, path, baseBranch)
}

// WorktreeAddExisting creates a worktree for an existing branch
func (g *Git) WorktreeAddExisting(path, branch string) (*Result, error) {
	return g.run("worktree", "add", path, branch)
}

// WorktreeRemove removes a worktree
func (g *Git) WorktreeRemove(path string, force bool) (*Result, error) {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	return g.run(args...)
}

// WorktreePrune prunes worktree information
func (g *Git) WorktreePrune() (*Result, error) {
	return g.run("worktree", "prune")
}

// WorktreeList lists all worktrees
func (g *Git) WorktreeList() ([]*Worktree, error) {
	result, err := g.run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("git worktree list failed: %s", result.Stderr)
	}

	return parseWorktreeList(result.Stdout), nil
}

// Worktree represents a Git worktree
type Worktree struct {
	Path   string
	Branch string
	Commit string
}

// parseWorktreeList parses the output of git worktree list --porcelain
func parseWorktreeList(output string) []*Worktree {
	var worktrees []*Worktree
	var current *Worktree

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "branch ") && current != nil {
			current.Branch = strings.TrimPrefix(line, "branch ")
		} else if strings.HasPrefix(line, "HEAD ") && current != nil {
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		}
	}

	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// Fetch fetches from remote
func (g *Git) Fetch(remote string, refspec string) (*Result, error) {
	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
		if refspec != "" {
			args = append(args, refspec)
		}
	}
	return g.run(args...)
}

// FetchUnshallow converts a shallow clone to a full clone
func (g *Git) FetchUnshallow() (*Result, error) {
	return g.run("fetch", "--unshallow")
}

// SubmoduleUpdate updates submodules
func (g *Git) SubmoduleUpdate() (*Result, error) {
	return g.run("submodule", "update", "--init", "--recursive")
}

// Rebase performs a rebase
func (g *Git) Rebase(branch string) (*Result, error) {
	return g.run("rebase", branch)
}

// Merge performs a merge
func (g *Git) Merge(branch string) (*Result, error) {
	return g.run("merge", branch)
}

// Switch switches to a branch or detaches HEAD
func (g *Git) Switch(ref string, detach bool) (*Result, error) {
	args := []string{"switch"}
	if detach {
		args = append(args, "--detach")
	}
	args = append(args, ref)
	return g.run(args...)
}

// SwitchInWorktree switches branches in a specific worktree
func (g *Git) SwitchInWorktree(worktreePath, ref string, detach bool) (*Result, error) {
	// Create a new Git instance for the worktree
	wtGit := New(worktreePath, g.verbose)
	return wtGit.Switch(ref, detach)
}

// BranchExists checks if a branch exists
func (g *Git) BranchExists(branch string) (bool, error) {
	result, err := g.run("rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	return result.ExitCode == 0, nil
}

// IsBranchCheckedOut checks if a branch is checked out in any worktree
func (g *Git) IsBranchCheckedOut(branch string) (bool, string, error) {
	worktrees, err := g.WorktreeList()
	if err != nil {
		return false, "", err
	}

	for _, wt := range worktrees {
		if wt.Branch == "refs/heads/"+branch {
			return true, wt.Path, nil
		}
	}

	return false, "", nil
}

// Add stages files
func (g *Git) Add(pathspec string) (*Result, error) {
	return g.run("add", pathspec)
}

// Commit creates a commit
func (g *Git) Commit(message string, all bool, signoff bool, gpgSign bool) (*Result, error) {
	args := []string{"commit", "-m", message}
	if all {
		args = append(args, "--all")
	}
	if signoff {
		args = append(args, "--signoff")
	}
	if gpgSign {
		args = append(args, "--gpg-sign")
	}
	return g.run(args...)
}

// Push pushes to remote
func (g *Git) Push(remote, branch string, setUpstream bool, force bool) (*Result, error) {
	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u")
	}
	if force {
		args = append(args, "--force")
	}
	args = append(args, remote, branch)
	return g.run(args...)
}

// RevParse runs git rev-parse
func (g *Git) RevParse(ref string) (string, error) {
	result, err := g.run("rev-parse", ref)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("git rev-parse failed: %s", result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// Status returns git status output
func (g *Git) Status() (*Result, error) {
	return g.run("status")
}

// CreatePRWithGH creates a pull request using gh CLI
func (g *Git) CreatePRWithGH(title, body, base string) (*Result, error) {
	return g.run("gh", "pr", "create", "--title", title, "--body", body, "--base", base)
}

// CreateMRWithGLab creates a merge request using glab CLI
func (g *Git) CreateMRWithGLab(title, description, targetBranch string) (*Result, error) {
	return g.run("glab", "mr", "create", "--title", title, "--description", description, "--target-branch", targetBranch)
}

// CurrentBranch returns the current branch name
func (g *Git) CurrentBranch() (string, error) {
	result, err := g.run("branch", "--show-current")
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("git branch --show-current failed: %s", result.Stderr)
	}
	return result.Stdout, nil
}
