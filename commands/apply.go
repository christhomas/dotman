package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dotman/services"

	"github.com/spf13/cobra"
)

func NewApplyCommand(dotman *services.DotmanService, fs *services.FileService) *cobra.Command {
	var dryRun bool
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
			userHome := fs.HomeDir()

			type fileDiff struct {
				RelPath  string
				RepoHash string
				UserHash string
				RepoDate string
				UserDate string
			}
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

			fmt.Print("Apply these changes to your home directory? [y/N]: ")
			if scan := bufio.NewScanner(os.Stdin); scan.Scan() {
				resp := strings.ToLower(strings.TrimSpace(scan.Text()))
				if resp != "y" && resp != "yes" {
					fmt.Println("[apply] Aborted.")
					return
				}
			}

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

			for _, info := range toCreate {
				src := filepath.Join(repoHome, info.RelPath)
				dst := filepath.Join(userHome, info.RelPath)
				repoStat, err := os.Stat(src)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[apply] Failed to stat %s: %v\n", src, err)
					continue
				}
				if err := fs.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					fmt.Fprintf(os.Stderr, "[apply] Failed to create directory for %s: %v\n", dst, err)
					continue
				}
				if err := fs.CopyFile(src, dst, repoStat.Mode()); err != nil {
					fmt.Fprintf(os.Stderr, "[apply] Failed to copy %s: %v\n", info.RelPath, err)
					continue
				}
				fmt.Printf("[apply] Created %s\n", info.RelPath)
			}
			for _, info := range toUpdate {
				src := filepath.Join(repoHome, info.RelPath)
				dst := filepath.Join(userHome, info.RelPath)
				repoStat, err := os.Stat(src)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[apply] Failed to stat %s: %v\n", src, err)
					continue
				}
				if err := fs.CopyFile(src, dst, repoStat.Mode()); err != nil {
					fmt.Fprintf(os.Stderr, "[apply] Failed to update %s: %v\n", info.RelPath, err)
					continue
				}
				fmt.Printf("[apply] Updated %s\n", info.RelPath)
			}
			fmt.Printf("[apply] Applied %d new file(s), updated %d file(s) in home directory.\n", len(toCreate), len(toUpdate))
		},
	}
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Run apply without making changes")
	return cmd
}
