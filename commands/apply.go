package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dotman/services"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
)

type fileDiff struct {
	RelPath  string
	RepoHash string
	UserHash string
	RepoDate string
	UserDate string
}

func NewApplyCommand(dotman *services.DotmanService, git *services.GitService, fs *services.FileService) *cobra.Command {
	var dryRun bool
	var noPull bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply dotfiles to your home directory",
		Run: func(cmd *cobra.Command, args []string) {
			if _, err := dotman.IsInitialized(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			repoHome, err := dotman.GetHomeDir()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			if !noPull {
				if dryRun {
					fmt.Println("[apply] Dry run: would pull with rebase from remote.")
				} else {
					pullOut, err := git.PullRebase(repoHome)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[apply] Pull failed: %v\n%s", err, pullOut)
						os.Exit(1)
					}
					fmt.Println("[apply] Pulled latest changes from remote.")
				}
			}

			userHome := fs.HomeDir()

			var toUpdate []fileDiff
			var toCreate []fileDiff

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
				if userHash == "missing" {
					toCreate = append(toCreate, fileDiff{
						RelPath:  relPath,
						RepoHash: repoHash,
						UserHash: userHash,
						RepoDate: repoDate,
						UserDate: userDate,
					})
				} else if repoHash != userHash {
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
				fmt.Fprintf(os.Stderr, "[apply] Error scanning files: %v\n", err)
				os.Exit(1)
			}

			if len(toCreate) > 0 {
				fmt.Println("[apply] The following files are missing and will be created:")
				for _, info := range toCreate {
					fmt.Printf("  - %s\n", info.RelPath)
				}
			}

			if len(toUpdate) > 0 {
				fmt.Println("[apply] The following files are different and can be updated:")
				for _, info := range toUpdate {
					fmt.Printf("  - %s\n    repo: %s (%s)\n    user: %s (%s)\n", info.RelPath, info.RepoHash, info.RepoDate, info.UserHash, info.UserDate)
				}
			}

			if len(toCreate) == 0 && len(toUpdate) == 0 {
				fmt.Println("[apply] No files to apply.")
				return
			}

			for {
				fmt.Print("Apply these changes to your home directory? [y/N/d]: ")
				scan := bufio.NewScanner(os.Stdin)
				if !scan.Scan() {
					fmt.Println("[apply] Aborted.")
					return
				}
				resp := strings.ToLower(strings.TrimSpace(scan.Text()))
				switch resp {
				case "y", "yes":
					if dryRun {
						fmt.Println("[apply] Dry run: would copy the following files:")
						for _, info := range toCreate {
							fmt.Printf("  - %s\n", info.RelPath)
						}
						for _, info := range toUpdate {
							fmt.Printf("  - %s\n", info.RelPath)
						}
						return
					}

					applyFiles(fs, toCreate, repoHome, userHome)
					applyFiles(fs, toUpdate, repoHome, userHome)
					fmt.Printf("[apply] Applied %d new file(s), updated %d file(s) in home directory.\n", len(toCreate), len(toUpdate))
					return
				case "n", "no", "":
					fmt.Println("[apply] Aborted.")
					return
				case "d", "diff":
					showDifferences(toUpdate, repoHome, userHome)
					continue // re-prompt
				default:
					fmt.Println("[apply] Please enter 'y', 'n', or 'd'.")
				}
			}
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Run apply without making changes")
	cmd.Flags().BoolVar(&noPull, "no-pull", false, "Skip git pull before applying changes")
	return cmd
}

func showDifferences(files []fileDiff, repoHome, userHome string) {
	for _, info := range files {
		repoPath := filepath.Join(repoHome, info.RelPath)
		userPath := filepath.Join(userHome, info.RelPath)
		repoContent, err1 := os.ReadFile(repoPath)
		userContent, err2 := os.ReadFile(userPath)
		if err1 != nil || err2 != nil {
			fmt.Printf("[diff] Error reading files for %s\n", info.RelPath)
			continue
		}
		ud := difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(userContent)),
			B:        difflib.SplitLines(string(repoContent)),
			FromFile: userPath,
			ToFile:   repoPath,
			Context:  3,
		}
		diffText, err := difflib.GetUnifiedDiffString(ud)
		if err != nil {
			fmt.Printf("[diff] Error generating diff for %s: %v\n", info.RelPath, err)
			continue
		}
		fmt.Printf("\n[diff] %s\n", info.RelPath)
		for _, line := range strings.Split(diffText, "\n") {
			switch {
			case strings.HasPrefix(line, "+"):
				fmt.Printf("\033[32m%s\033[0m\n", line)
			case strings.HasPrefix(line, "-"):
				fmt.Printf("\033[31m%s\033[0m\n", line)
			default:
				fmt.Println(line)
			}
		}

	}
}

func applyFiles(fs *services.FileService, files []fileDiff, repoHome, userHome string) {
	for _, info := range files {
		src := filepath.Join(repoHome, info.RelPath)
		dst := filepath.Join(userHome, info.RelPath)

		action := "create"
		if _, err := os.Stat(dst); err != nil {
			action = "update"
		}

		repoStat, err := os.Stat(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[apply] Failed to stat input file %s: %v\n", src, err)
			continue
		}
		if err := fs.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[apply] Failed to create directory for %s: %v\n", dst, err)
			continue
		}
		if err := fs.CopyFile(src, dst, repoStat.Mode()); err != nil {
			fmt.Fprintf(os.Stderr, "[apply] Failed to %s %s: %v\n", action, info.RelPath, err)
			continue
		}
		fmt.Printf("[apply] %s %s\n", action, info.RelPath)
	}
}
