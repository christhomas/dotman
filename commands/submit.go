package commands

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dotman/services"

	"github.com/spf13/cobra"
)

type fileDiff struct {
	RelPath  string
	RepoHash string
	UserHash string
	RepoDate string
	UserDate string
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func shortUniquePrefix(a, b string) (string, string) {
	minLen := 7
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	for l := minLen; l <= maxLen; l++ {
		if len(a) >= l && len(b) >= l && a[:l] != b[:l] {
			return a[:l], b[:l]
		}
	}
	return a, b
}

func NewSubmitCommand(dotman *services.DotmanService, git *services.GitService, publishCmd *cobra.Command, fs *services.FileService) *cobra.Command {
	var verbose bool
	var publish bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Copy modified tracked files from home into the dotman repo and commit them",
		Run: func(cmd *cobra.Command, args []string) {
			repoDir, err := dotman.IsInitialized()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			repoHome, err := dotman.GetHomeDir()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			userHome := fs.HomeDir()

			var toUpdate []fileDiff

			// 1. Detect content-changed files
			err = filepath.Walk(repoHome, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				relPath, _ := filepath.Rel(repoHome, path)
				userFile := filepath.Join(userHome, relPath)
				repoHash, _ := fileHash(path)
				userHash := "missing"
				repoDate := "missing"
				userDate := "missing"
				if stat, err := os.Stat(path); err == nil {
					repoDate = stat.ModTime().Format("2006-01-02 15:04:05")
				}
				if stat, err := os.Stat(userFile); err == nil {
					userHash, _ = fileHash(userFile)
					userDate = stat.ModTime().Format("2006-01-02 15:04:05")
				}
				if repoHash != "missing" && userHash != "missing" {
					repoHash, userHash = shortUniquePrefix(repoHash, userHash)
				}
				if repoHash != userHash && userHash != "missing" {
					toUpdate = append(toUpdate, fileDiff{
						RelPath:  relPath,
						RepoHash: repoHash,
						UserHash: userHash,
						RepoDate: repoDate,
						UserDate: userDate,
					})
				}
				return nil
			})

			if err != nil {
				fmt.Fprintln(os.Stderr, "[submit] Error scanning files:", err)
				os.Exit(1)
			}

			// Gather both content-changed files (toUpdate) and uncommitted/untracked files (git.Status)
			statusFiles, err := git.Status(repoHome)
			if err != nil {
				fmt.Fprintln(os.Stderr, "[submit] Failed to check git status:", err)
				os.Exit(1)
			}

			// Build a set of all files to submit (union of relPaths from toUpdate and statusFiles)
			fileSet := make(map[string]struct{})
			for _, info := range toUpdate {
				fileSet[info.RelPath] = struct{}{}
			}
			for _, f := range statusFiles {
				fileSet[f] = struct{}{}
			}
			if len(fileSet) == 0 {
				fmt.Println("[submit] No changed files to submit.")
				return
			}

			// Show a single summary of all files to submit
			fmt.Println("[submit] The following files will be submitted:")
			for f := range fileSet {
				fmt.Printf("  - %s\n", f)
			}

			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Proceed to copy, stage, and commit these files? [y/N]: ")
			respRaw, _ := reader.ReadString('\n')
			resp := strings.ToLower(strings.TrimSpace(respRaw))
			if resp != "y" && resp != "yes" {
				fmt.Println("[submit] Aborted.")
				return
			}

			if dryRun {
				fmt.Println("[submit] Dry run: would copy and commit the following files:")
				for f := range fileSet {
					fmt.Printf("  - %s\n", f)
				}
				return
			}

			// Copy changed files from $HOME to repo (only those in toUpdate)
			for _, info := range toUpdate {
				src := filepath.Join(userHome, info.RelPath)
				dst := filepath.Join(repoHome, info.RelPath)
				userStat, err := os.Stat(src)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[submit] Skipping %s (missing in $HOME)\n", info.RelPath)
					continue
				}
				if err := fs.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					fmt.Fprintf(os.Stderr, "[submit] Failed to create directory for %s: %v\n", dst, err)
					continue
				}
				if err := fs.CopyFile(src, dst, userStat.Mode()); err != nil {
					fmt.Fprintf(os.Stderr, "[submit] Failed to copy %s: %v\n", info.RelPath, err)
					continue
				}
				fmt.Printf("[submit] Copied %s\n", info.RelPath)
			}

			// Stage all files (some may not exist in $HOME, but are tracked/uncommitted)
			var allRelPaths []string
			for f := range fileSet {
				allRelPaths = append(allRelPaths, f)
			}

			git.SetVerbose(verbose)
			if err := git.Add(repoDir, allRelPaths); err != nil {
				fmt.Fprintf(os.Stderr, "[submit] Failed to stage files: %v\n", err)
				os.Exit(1)
			}

			fmt.Print("Commit message (leave blank for default): ")
			commitMsg := "Update dotfiles"
			msgRaw, _ := reader.ReadString('\n')
			msg := strings.TrimSpace(msgRaw)
			if msg != "" {
				commitMsg = msg
			}
			if err := git.Commit(repoHome, commitMsg); err != nil {
				fmt.Fprintf(os.Stderr, "[submit] Failed to commit: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[submit] Committed %d file(s).\n", len(allRelPaths))
			if publish {
				publishCmd.Flags().Set("no-pull", "false")
				if verbose {
					publishCmd.Flags().Set("verbose", "true")
				}
				publishCmd.Run(cmd, args)
			}
		},
	}

	cmd.Flags().BoolVar(&publish, "publish", false, "Publish after submitting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without committing")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	return cmd
}
