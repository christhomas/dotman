package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dotman/services"
	"github.com/spf13/cobra"
)

func NewShowCommand(dotman *services.DotmanService) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show managed files in a tree view",
		Run: func(cmd *cobra.Command, args []string) {
			runShow(dotman)
		},
	}
}

func runShow(dotman *services.DotmanService) {
	repoRoot, err := dotman.IsInitialized()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoHome := filepath.Join(repoRoot, "home")
	userHome, _ := os.UserHomeDir()

	rootLabel := fmt.Sprintf("home (repo: %s → extracts to %s)", repoHome, userHome)

	lines, err := renderTree(repoHome, rootLabel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[status] Failed to render tree: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(strings.Join(lines, "\n"))
}

func renderTree(rootPath, label string) ([]string, error) {
	lines := []string{label}

	var walk func(path, prefix string) error
	walk = func(path, prefix string) error {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		// filter out common non-managed dirs
		filtered := entries[:0]
		for _, e := range entries {
			if e.Name() == ".git" || e.Name() == ".history" {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered

		for i, e := range entries {
			connector := "├── "
			nextPrefix := prefix + "│   "
			if i == len(entries)-1 {
				connector = "└── "
				nextPrefix = prefix + "    "
			}
			line := fmt.Sprintf("%s%s%s", prefix, connector, e.Name())
			lines = append(lines, line)
			if e.IsDir() {
				if err := walk(filepath.Join(path, e.Name()), nextPrefix); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(rootPath, ""); err != nil {
		return nil, err
	}
	return lines, nil
}
