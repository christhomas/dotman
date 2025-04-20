package main

import (
	"os"

	"dotman/commands"
	"dotman/services"

	"github.com/spf13/cobra"
)

func main() {
	// CLI root command
	rootCmd := &cobra.Command{
		Use:   "dotman",
		Short: "Dotman is a dotfile manager",
		Long:  `Dotman - transparent, Git-backed dotfile workflow manager`,
	}

	// Register all subcommands directly
	dotman := services.NewDotmanService()
	fs := services.NewFileService()
	git := services.NewGitService()
	cfg := services.NewConfigService()

	publishCmd := commands.NewPublishCommand(dotman, git)

	rootCmd.AddCommand(
		commands.NewAddCommand(dotman, fs),
		commands.NewConfigCommand(cfg),
		publishCmd,
		commands.NewSubmitCommand(dotman, git, publishCmd, fs),
		commands.NewBootstrapCommand(dotman, fs),
		commands.NewApplyCommand(dotman, fs),
		commands.NewInitCommand(dotman, git, cfg),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
