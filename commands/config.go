package commands

import (
	"fmt"
	"os"
	"dotman/services"
	"github.com/spf13/cobra"
)

func NewConfigCommand(cfg *services.ConfigService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get or set dotman configuration",
		Long:  `Get or set values in ~/.dotman.json using dot notation.`,
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := cfg.Load(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
				os.Exit(1)
			}
			val, err := cfg.Get(args[0])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Key not found: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("%v\n", val)
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if err := cfg.Load(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
				os.Exit(1)
			}
			if err := cfg.Set(args[0], args[1]); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to set value: %v\n", err)
				os.Exit(1)
			}
			if err := cfg.Save(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("OK")
		},
	}

	cmd.AddCommand(getCmd)
	cmd.AddCommand(setCmd)
	return cmd
}
