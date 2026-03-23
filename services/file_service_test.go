package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareFiles_NoChanges(t *testing.T) {
	t.Parallel()

	repoHome := t.TempDir()
	userHome := t.TempDir()

	// Same content in both
	writeFile(t, filepath.Join(repoHome, ".zshrc"), "export FOO=bar")
	writeFile(t, filepath.Join(userHome, ".zshrc"), "export FOO=bar")

	fs := NewFileService()
	changed, created, err := fs.CompareFiles(repoHome, userHome)
	if err != nil {
		t.Fatalf("CompareFiles: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("expected 0 changed, got %d", len(changed))
	}
	if len(created) != 0 {
		t.Fatalf("expected 0 created, got %d", len(created))
	}
}

func TestCompareFiles_Changed(t *testing.T) {
	t.Parallel()

	repoHome := t.TempDir()
	userHome := t.TempDir()

	writeFile(t, filepath.Join(repoHome, ".zshrc"), "export FOO=bar")
	writeFile(t, filepath.Join(userHome, ".zshrc"), "export FOO=baz")

	fs := NewFileService()
	changed, created, err := fs.CompareFiles(repoHome, userHome)
	if err != nil {
		t.Fatalf("CompareFiles: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(changed))
	}
	if changed[0].RelPath != ".zshrc" {
		t.Fatalf("expected .zshrc, got %s", changed[0].RelPath)
	}
	if len(created) != 0 {
		t.Fatalf("expected 0 created, got %d", len(created))
	}
}

func TestCompareFiles_Created(t *testing.T) {
	t.Parallel()

	repoHome := t.TempDir()
	userHome := t.TempDir()

	// File exists in repo but not in user home
	writeFile(t, filepath.Join(repoHome, ".vimrc"), "set nocompatible")

	fs := NewFileService()
	changed, created, err := fs.CompareFiles(repoHome, userHome)
	if err != nil {
		t.Fatalf("CompareFiles: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("expected 0 changed, got %d", len(changed))
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(created))
	}
	if created[0].RelPath != ".vimrc" {
		t.Fatalf("expected .vimrc, got %s", created[0].RelPath)
	}
}

func TestCompareFiles_Nested(t *testing.T) {
	t.Parallel()

	repoHome := t.TempDir()
	userHome := t.TempDir()

	// Nested file with changes
	repoDir := filepath.Join(repoHome, ".config", "nvim")
	userDir := filepath.Join(userHome, ".config", "nvim")
	os.MkdirAll(repoDir, 0755)
	os.MkdirAll(userDir, 0755)

	writeFile(t, filepath.Join(repoDir, "init.lua"), "-- repo version")
	writeFile(t, filepath.Join(userDir, "init.lua"), "-- local version")

	fs := NewFileService()
	changed, _, err := fs.CompareFiles(repoHome, userHome)
	if err != nil {
		t.Fatalf("CompareFiles: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(changed))
	}
	if changed[0].RelPath != ".config/nvim/init.lua" {
		t.Fatalf("expected .config/nvim/init.lua, got %s", changed[0].RelPath)
	}
}

func TestCompareFiles_Mixed(t *testing.T) {
	t.Parallel()

	repoHome := t.TempDir()
	userHome := t.TempDir()

	// Unchanged
	writeFile(t, filepath.Join(repoHome, ".bashrc"), "same")
	writeFile(t, filepath.Join(userHome, ".bashrc"), "same")

	// Changed
	writeFile(t, filepath.Join(repoHome, ".zshrc"), "repo")
	writeFile(t, filepath.Join(userHome, ".zshrc"), "local")

	// Missing from user (created)
	writeFile(t, filepath.Join(repoHome, ".gitconfig"), "new file")

	fs := NewFileService()
	changed, created, err := fs.CompareFiles(repoHome, userHome)
	if err != nil {
		t.Fatalf("CompareFiles: %v", err)
	}
	if len(changed) != 1 {
		t.Fatalf("expected 1 changed, got %d", len(changed))
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 created, got %d", len(created))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile(%s): %v", path, err)
	}
}
