package commands

import (
	"dotman/services"
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

func NewInitCommand(dotman *services.DotmanService, git *services.GitService, cfg *services.ConfigService) *cobra.Command {
	fs := services.NewFileService()
	cmd := &cobra.Command{
		Use:   "init [repourl] <folderpath>",
		Short: "Initialize dotman repository",
		Long: ` 
Initialize dotman in a folder, optionally cloning a repo.

Examples:
  dotman init ~/dotfiles
  dotman init https://github.com/user/dotfiles.git ~/dotfiles`,
		Args: cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			_ = cfg.Load()
			if len(args) == 1 {
				// dotman init <folderpath>
				fmt.Printf("Initializing dotman in existing folder: %s\n", args[0])
				dotman := services.NewDotmanService()
				absPath := fs.ExpandHome(args[0])
				// Canonicalize and validate dotfile path using DotmanService logic
				if abs, err := dotman.CanonicalizePath(absPath); err == nil {
					absPath = abs
				}
				cfg.Set("dotfile.path", absPath)
				_ = cfg.Save()
				fmt.Println("Initialized dotman in existing repo.")
			} else if len(args) == 2 {
				// dotman init <repourl> <folderpath>
				fmt.Printf("Cloning %s into %s...\n", args[0], args[1])
				target := fs.ExpandHome(args[1])
				if err := git.CloneRepo(args[0], target, false, false); err != nil {
					fmt.Fprintf(os.Stderr, "Git clone failed: %v\n", err)
					os.Exit(1)
				}
				dotman := services.NewDotmanService()
				absPath := target
				// Canonicalize and validate dotfile path using DotmanService logic
				if abs, err := dotman.CanonicalizePath(absPath); err == nil {
					absPath = abs
				}
				cfg.Set("dotfile.path", absPath)
				_ = cfg.Save()
				fmt.Println("Initialized dotman in cloned repo.")
				// TODO: initialize config/repo in args[1]
			} else {
				fmt.Println("Usage: dotman init [repourl] <folderpath>")
			}
		},
	}
	return cmd
}
