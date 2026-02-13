package git

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"

	"github.com/kernel-labs-ai/awt/internal/logger"
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

// runExternal executes a non-git command (e.g., gh, glab) with the working directory set to the worktree root
func (g *Git) runExternal(name string, args ...string) (*Result, error) {
	if g.verbose {
		logger.Debug("%s %s", name, strings.Join(args, " "))
	}

	cmd := exec.Command(name, args...)
	cmd.Dir = g.workTreeRoot

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
			return result, fmt.Errorf("failed to execute %s: %w", name, err)
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
	return g.runExternal("gh", "pr", "create", "--title", title, "--body", body, "--base", base)
}

// CreateMRWithGLab creates a merge request using glab CLI
func (g *Git) CreateMRWithGLab(title, description, targetBranch string) (*Result, error) {
	return g.runExternal("glab", "mr", "create", "--title", title, "--description", description, "--target-branch", targetBranch)
}

// GetRemoteURL returns the URL for a given remote
func (g *Git) GetRemoteURL(remote string) (string, error) {
	result, err := g.run("remote", "get-url", remote)
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to get remote URL: %s", result.Stderr)
	}
	return result.Stdout, nil
}

// remoteInfo holds the parsed host, owner, and repo from a remote URL
type remoteInfo struct {
	Host  string
	Owner string
	Repo  string
}

// sshRemotePattern matches SSH remote URLs like git@github.com:owner/repo.git
var sshRemotePattern = regexp.MustCompile(`^[\w.-]+@([\w.-]+):([\w._-]+)/([\w._-]+?)(?:\.git)?$`)

// parseRemoteURL parses a git remote URL (SSH or HTTPS) into host, owner, and repo
func parseRemoteURL(rawURL string) (*remoteInfo, error) {
	rawURL = strings.TrimSpace(rawURL)

	// Try SSH format: git@github.com:owner/repo.git
	if matches := sshRemotePattern.FindStringSubmatch(rawURL); matches != nil {
		return &remoteInfo{
			Host:  matches[1],
			Owner: matches[2],
			Repo:  matches[3],
		}, nil
	}

	// Try HTTPS format: https://github.com/owner/repo.git
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote URL %q: %w", rawURL, err)
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("could not parse owner/repo from URL %q", rawURL)
	}

	repo := parts[1]
	repo = strings.TrimSuffix(repo, ".git")

	return &remoteInfo{
		Host:  parsed.Host,
		Owner: parts[0],
		Repo:  repo,
	}, nil
}

// CompareURL constructs a browser-openable compare URL for creating a PR/MR
func (g *Git) CompareURL(remote, branch, base string) (string, error) {
	remoteURL, err := g.GetRemoteURL(remote)
	if err != nil {
		return "", err
	}

	info, err := parseRemoteURL(remoteURL)
	if err != nil {
		return "", err
	}

	switch {
	case strings.Contains(info.Host, "github"):
		return fmt.Sprintf("https://%s/%s/%s/compare/%s...%s?expand=1", info.Host, info.Owner, info.Repo, base, branch), nil
	case strings.Contains(info.Host, "gitlab"):
		return fmt.Sprintf("https://%s/%s/%s/-/merge_requests/new?merge_request[source_branch]=%s&merge_request[target_branch]=%s", info.Host, info.Owner, info.Repo, branch, base), nil
	default:
		return fmt.Sprintf("https://%s/%s/%s/compare/%s...%s", info.Host, info.Owner, info.Repo, base, branch), nil
	}
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

// SetUpstream sets the upstream tracking branch for the current branch
// This uses git config to set the tracking, which works even if the remote branch doesn't exist yet
func (g *Git) SetUpstream(remote, branch string) (*Result, error) {
	// Get current branch first
	currentBranch, err := g.CurrentBranch()
	if err != nil {
		return &Result{ExitCode: 1, Stderr: fmt.Sprintf("failed to get current branch: %v", err)}, err
	}

	// Set the upstream using git config (works even if remote branch doesn't exist)
	// The merge ref should point to refs/heads/<branch>, not refs/remotes/<remote>/<branch>
	// Git will automatically map this to the remote-tracking branch via the remote's fetch spec
	upstreamRef := fmt.Sprintf("refs/heads/%s", branch)
	configKey := fmt.Sprintf("branch.%s.remote", currentBranch)
	configMergeKey := fmt.Sprintf("branch.%s.merge", currentBranch)

	// Set remote
	remoteResult, err := g.run("config", configKey, remote)
	if err != nil || remoteResult.ExitCode != 0 {
		return remoteResult, err
	}

	// Set merge ref
	mergeResult, err := g.run("config", configMergeKey, upstreamRef)
	return mergeResult, err
}
