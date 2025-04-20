package commands

import (
	"fmt"
	"os"

	"dotman/services"

	"github.com/spf13/cobra"
)

func NewPublishCommand(dotman *services.DotmanService, git *services.GitService) *cobra.Command {
	var noPull bool
	var dryRun bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Pull and push the dotman repository to sync with remote",
		Run: func(cmd *cobra.Command, args []string) {
			git.SetVerbose(verbose)
			if _, err := dotman.IsInitialized(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			repoHomeDir, err := dotman.GetHomeDir()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}

			if !noPull {
				if dryRun {
					fmt.Println("[publish] Dry run: would pull with rebase from remote.")
				} else {
					pullOut, err := git.PullRebase(repoHomeDir)
					if err != nil {
						fmt.Fprintf(os.Stderr, "[publish] Pull failed: %v\n%s", err, pullOut)
						os.Exit(1)
					}
					fmt.Println("[publish] Pulled latest changes from remote.")
				}
			}

			if dryRun {
				fmt.Println("[publish] Dry run: would push changes to remote.")
				return
			}

			pushOut, err := git.Push(repoHomeDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[publish] Push failed: %v\n%s", err, pushOut)
				os.Exit(1)
			}
			fmt.Println("[publish] Dotfiles updated on remote.")
		},
	}
	cmd.Flags().BoolVar(&noPull, "no-pull", false, "Skip pull step and only push")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview publish without making changes")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose git output")
	return cmd
}
