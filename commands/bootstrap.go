package commands

import (
	"dotman/services"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func NewBootstrapCommand(dotman *services.DotmanService, fs *services.FileService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Run the bootstrap script from your dotfiles repo",
		Run: func(cmd *cobra.Command, args []string) {
			dotman := services.NewDotmanService()
			dir, err := dotman.IsInitialized()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			hookPath := dir + "/hooks/bootstrap.sh"
			bsPath := dir + "/bootstrap.sh"
			scriptToRun := ""
			if err := fs.Exists(hookPath); err == nil {
				scriptToRun = hookPath
			} else if err := fs.Exists(bsPath); err == nil {
				scriptToRun = bsPath
			} else {
				fmt.Fprintf(os.Stderr, "[ERROR] No bootstrap.sh found in hooks/ or root of %s\n", dir)
				os.Exit(1)
			}
			cmdExec := exec.Command("bash", scriptToRun)
			cmdExec.Stdout = os.Stdout
			cmdExec.Stderr = os.Stderr
			cmdExec.Stdin = os.Stdin
			fmt.Printf("[INFO] Running %s...\n", scriptToRun)
			if err := cmdExec.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Bootstrap failed: %v\n", err)
				os.Exit(1)
			}
		},
	}
	return cmd
}
