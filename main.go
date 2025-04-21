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

	commandList := make(map[string]*cobra.Command)
	commandList["init"] = commands.NewInitCommand(dotman, git, cfg)
	commandList["bootstrap"] = commands.NewBootstrapCommand(dotman, fs)
	commandList["apply"] = commands.NewApplyCommand(dotman, git, fs)
	commandList["publish"] = commands.NewPublishCommand(dotman, git)
	commandList["submit"] = commands.NewSubmitCommand(dotman, git, commandList["publish"], fs)
	commandList["add"] = commands.NewAddCommand(dotman, fs)
	commandList["config"] = commands.NewConfigCommand(cfg)

	rootCmd.AddCommand(
		commandList["init"],
		commandList["bootstrap"],
		commandList["apply"],
		commandList["submit"],
		commandList["publish"],
		commandList["add"],
		commandList["config"],
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
