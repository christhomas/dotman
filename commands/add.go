package commands

import (
	"dotman/services"
	"fmt"

	"github.com/spf13/cobra"
)

func NewAddCommand(dotman *services.DotmanService, fs *services.FileService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [file]",
		Short: "Add a file from $HOME into the repo",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			file := args[0]
			dotmanDir, err := dotman.IsInitialized()
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), err)
				return
			}
			fs := services.NewFileService()
			homeDir := fs.HomeDir()
			srcPath := file
			if !fs.IsAbs(srcPath) {
				srcPath = fs.Join(homeDir, srcPath)
			}
			info, err := fs.Stat(srcPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ERROR] Source file does not exist: %s\n", srcPath)
				return
			}
			if info.IsDir() {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ERROR] Directories are not supported yet: %s\n", srcPath)
				return
			}
			relPath, err := fs.Rel(homeDir, srcPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ERROR] Failed to compute relative path: %v\n", err)
				return
			}
			destPath := fs.Join(dotmanDir, "home", relPath)
			if err := fs.MkdirAll(fs.Join(destPath, ".."), 0755); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ERROR] Failed to create destination directory: %v\n", err)
				return
			}
			if err := fs.CopyFile(srcPath, destPath, info.Mode()); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ERROR] Failed to copy file: %v\n", err)
				return
			}
			fmt.Fprintf(cmd.OutOrStdout(), "[INFO] Added %s to repo as %s\n", srcPath, destPath)
		},
	}
	return cmd
}
