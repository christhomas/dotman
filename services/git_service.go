package services

import (
	"fmt"
	"os/exec"
	"strings"
)

type GitService struct {
	verbose bool
}

func (g *GitService) ExecCommand(dir string, args ...string) *exec.Cmd {
	if g.verbose {
		fmt.Printf("[git] (%s) Running: git %s\n", dir, strings.Join(args, " "))
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd
}

// NewGitService creates a new GitService instance.
func NewGitService() *GitService {
	return &GitService{}
}

func (g *GitService) SetVerbose(v bool) {
	g.verbose = v
}

// Status returns a list of changed tracked files (modified or untracked) in the repo at dir.
func (g *GitService) Status(dir string) ([]string, error) {
	cmd := g.ExecCommand(dir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, line := range lines {
		if len(line) >= 4 && (line[:2] == " M" || line[:2] == "??") {
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
}

// Add stages the given files in the repo at dir.
func (g *GitService) Add(dir string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add"}, files...)
	cmd := g.ExecCommand(dir, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %w\n%s", err, string(out))
	}
	return nil
}

// Commit creates a commit with the given message in the repo at dir.
func (g *GitService) Commit(dir, message string) error {
	cmd := g.ExecCommand(dir, "commit", "-m", message)
	return cmd.Run()
}

// PullRebase performs a git pull --rebase in the repo at dir and returns output and error.
func (g *GitService) PullRebase(dir string) ([]byte, error) {
	cmd := g.ExecCommand(dir, "pull", "--rebase")
	return cmd.CombinedOutput()
}

// Push performs a git push in the repo at dir and returns output and error.
func (g *GitService) Push(dir string) ([]byte, error) {
	cmd := g.ExecCommand(dir, "push")
	return cmd.CombinedOutput()
}

// CloneRepo clones a git repo to the target directory.
// If depth1 is true, does a shallow clone. If noCheckout is true, disables checkout.
func (g *GitService) CloneRepo(repoURL, targetDir string, depth1, noCheckout bool) error {
	args := []string{"clone"}
	if depth1 {
		args = append(args, "--depth", "1")
	}
	if noCheckout {
		args = append(args, "--no-checkout")
	}
	args = append(args, repoURL, targetDir)
	cmd := g.ExecCommand("", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// IsRemoteGitRepo returns true if the given URL is a valid git repository.
func (g *GitService) IsRemoteGitRepo(url string) bool {
	if len(url) == 0 {
		return false
	}
	cmd := g.ExecCommand("", "ls-remote", url)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
