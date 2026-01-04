package commands

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dotman/diffview"
	"dotman/services"
	"dotman/types"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func shortUniquePrefix(a, b string) (string, string) {
	minLen := 7
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	for l := minLen; l <= maxLen; l++ {
		if len(a) >= l && len(b) >= l && a[:l] != b[:l] {
			return a[:l], b[:l]
		}
	}
	return a, b
}

func normalizeRelPath(rel string) string {
	rel = strings.TrimPrefix(rel, "./")
	if strings.HasPrefix(rel, "home/") {
		rel = strings.TrimPrefix(rel, "home/")
	}
	return rel
}

type commitModel struct {
	input      textinput.Model
	defaultMsg string
}

func (m *commitModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *commitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			m.input.SetValue("")
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *commitModel) View() string {
	var b strings.Builder
	b.WriteString("\nEnter commit message (Esc for default):\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n")
	return b.String()
}

func promptCommitMessage(defaultMsg string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = defaultMsg
	ti.Focus()
	ti.CharLimit = 256
	ti.Prompt = "  > "

	p := tea.NewProgram(&commitModel{
		input:      ti,
		defaultMsg: defaultMsg,
	})
	res, err := p.Run()
	if err != nil {
		return "", err
	}
	m, ok := res.(*commitModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}
	msg := strings.TrimSpace(m.input.Value())
	if msg == "" {
		msg = defaultMsg
	}
	return msg, nil
}

type fileOption struct {
	label    string
	selected bool
}

type selectionModel struct {
	items    []fileOption
	cursor   int
	quit     bool
	canceled bool
}

func (m selectionModel) Init() tea.Cmd { return nil }

func (m selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "a":
			// toggle all
			allSelected := true
			for _, it := range m.items {
				if !it.selected {
					allSelected = false
					break
				}
			}
			for i := range m.items {
				m.items[i].selected = !allSelected
			}
		case " ", "enter":
			if len(m.items) == 0 {
				m.quit = true
				return m, tea.Quit
			}
			// toggle current item on space
			if msg.String() == " " {
				m.items[m.cursor].selected = !m.items[m.cursor].selected
				return m, nil
			}
			// enter quits
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectionModel) View() string {
	var b strings.Builder
	b.WriteString("Select files to submit (↑/↓ or j/k, space to toggle, a to toggle all, Enter to confirm, Esc to cancel)\n\n")
	if len(m.items) == 0 {
		b.WriteString("No files available.\n")
		return b.String()
	}
	for i, it := range m.items {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		check := "[ ]"
		if it.selected {
			check = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s %s %s\n", cursor, check, it.label))
	}
	return b.String()
}

func startSubmitWizard(paths []string) ([]string, bool, error) {
	items := make([]fileOption, len(paths))
	for i, p := range paths {
		items[i] = fileOption{label: p, selected: true}
	}
	p := tea.NewProgram(selectionModel{items: items})
	res, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	m, ok := res.(selectionModel)
	if !ok {
		return nil, false, fmt.Errorf("unexpected model type")
	}
	if m.canceled {
		return nil, false, nil
	}
	var selected []string
	for _, it := range m.items {
		if it.selected {
			selected = append(selected, it.label)
		}
	}
	if len(selected) == 0 {
		return nil, false, nil
	}
	return selected, true, nil
}

func NewSubmitCommand(dotman *services.DotmanService, git *services.GitService, publishCmd *cobra.Command, fs *services.FileService) *cobra.Command {
	var verbose bool
	var publish bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Copy modified tracked files from home into the dotman repo and commit them",
		Run: func(cmd *cobra.Command, args []string) {
			runSubmit(cmd, args, dotman, git, publishCmd, fs, verbose, publish, dryRun)
		},
	}

	cmd.Flags().BoolVar(&publish, "publish", false, "Publish after submitting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without committing")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	return cmd
}

func runSubmit(cmd *cobra.Command, args []string, dotman *services.DotmanService, git *services.GitService, publishCmd *cobra.Command, fs *services.FileService, verbose, publish, dryRun bool) {
	repoDir, err := dotman.IsInitialized()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoHome, err := dotman.GetHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	userHome := fs.HomeDir()

	var toUpdate []types.FileDiff

	// 1. Detect content-changed files
	err = filepath.Walk(repoHome, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPathRaw, _ := filepath.Rel(repoHome, path)
		relPath := normalizeRelPath(relPathRaw)
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
		if repoHash != userHash && userHash != "missing" {
			toUpdate = append(toUpdate, types.FileDiff{
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
		fmt.Fprintln(os.Stderr, "[submit] Error scanning files:", err)
		os.Exit(1)
	}

	// Gather both content-changed files (toUpdate) and uncommitted/untracked files (git.Status)
	statusFiles, err := git.Status(repoHome)
	if err != nil {
		fmt.Fprintln(os.Stderr, "[submit] Failed to check git status:", err)
		os.Exit(1)
	}

	// Build a set of all files to submit (union of relPaths from toUpdate and statusFiles)
	fileSet := make(map[string]struct{})
	for _, info := range toUpdate {
		fileSet[info.RelPath] = struct{}{}
	}
	for _, f := range statusFiles {
		fileSet[normalizeRelPath(f)] = struct{}{}
	}
	if len(fileSet) == 0 {
		fmt.Println("[submit] No changed files to submit.")
		return
	}

	// Prepare a stable ordered list for the viewer
	var allRelPaths []string
	for f := range fileSet {
		allRelPaths = append(allRelPaths, f)
	}
	sort.Strings(allRelPaths)

	// Render diffs for each candidate file before prompting for selection.
	renderer := diffview.NewRenderer()
	for _, rel := range allRelPaths {
		panels, err := renderer.RenderFiles([]diffview.FilePair{{
			Label:     rel,
			LeftPath:  filepath.Join(repoHome, rel),
			RightPath: filepath.Join(userHome, rel),
		}}, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[submit] Failed to display diff viewer for %s: %v\n", rel, err)
			os.Exit(1)
		}
		for _, p := range panels {
			fmt.Println(p)
			fmt.Println()
		}
	}

	selectedPaths, proceed, err := startSubmitWizard(allRelPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[submit] Failed to select files: %v\n", err)
		os.Exit(1)
	}
	if !proceed {
		fmt.Println("[submit] No files selected. Aborting.")
		return
	}
	allRelPaths = selectedPaths

	if dryRun {
		fmt.Println("[submit] Dry run: would copy and commit the following files:")
		for _, f := range allRelPaths {
			fmt.Printf("  - %s\n", f)
		}
		return
	}

	// Copy changed files from $HOME to repo (only those in toUpdate)
	selectedSet := make(map[string]struct{}, len(allRelPaths))
	for _, f := range allRelPaths {
		selectedSet[f] = struct{}{}
	}
	for _, info := range toUpdate {
		if _, ok := selectedSet[normalizeRelPath(info.RelPath)]; !ok {
			continue
		}
		src := filepath.Join(userHome, normalizeRelPath(info.RelPath))
		dst := filepath.Join(repoHome, normalizeRelPath(info.RelPath))
		userStat, err := os.Stat(src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[submit] Skipping %s (missing in $HOME)\n", info.RelPath)
			continue
		}
		if err := fs.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[submit] Failed to create directory for %s: %v\n", dst, err)
			continue
		}
		if err := fs.CopyFile(src, dst, userStat.Mode()); err != nil {
			fmt.Fprintf(os.Stderr, "[submit] Failed to copy %s: %v\n", info.RelPath, err)
			continue
		}
		fmt.Printf("[submit] Copied %s\n", info.RelPath)
	}

	// Stage all files (some may not exist in $HOME, but are tracked/uncommitted)
	git.SetVerbose(verbose)
	stagePaths := make([]string, 0, len(allRelPaths))
	for _, rel := range allRelPaths {
		stagePaths = append(stagePaths, filepath.Join("home", rel))
	}
	if err := git.Add(repoDir, stagePaths); err != nil {
		fmt.Fprintf(os.Stderr, "[submit] Failed to stage files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Prepare commit message (Esc accepts default):")
	commitMsg, err := promptCommitMessage("Update dotfiles")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[submit] Failed to read commit message: %v\n", err)
		os.Exit(1)
	}
	if err := git.Commit(repoHome, commitMsg); err != nil {
		fmt.Fprintf(os.Stderr, "[submit] Failed to commit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[submit] Committed %d file(s).\n", len(allRelPaths))
	if publish {
		publishCmd.Flags().Set("no-pull", "false")
		if verbose {
			publishCmd.Flags().Set("verbose", "true")
		}
		publishCmd.Run(cmd, args)
	}
}
