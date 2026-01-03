package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DotmanService struct {
	Config *ConfigService
}

// CanonicalizePath returns an absolute, user-friendly path (e.g., ~/foo) given a path string.
func (d *DotmanService) CanonicalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	home, _ := os.UserHomeDir()
	if rel, err := filepath.Rel(home, absPath); err == nil && (rel == "." || !strings.HasPrefix(rel, "..")) {
		absPath = filepath.Join("~", rel)
	}
	return absPath, nil
}

func NewDotmanService() *DotmanService {
	cfg := NewConfigService()
	_ = cfg.Load()
	return &DotmanService{Config: cfg}
}

// IsInitialized checks if dotman is ready (config exists, dotfile.path set, directory exists)
// Returns (expandedDotfilePath, nil) if ready, ("", error) otherwise
func (d *DotmanService) IsInitialized() (string, error) {
	if err := d.Config.Load(); err != nil {
		return "", fmt.Errorf("[ERROR] Could not load dotman config. Have you initialized dotman?")
	}
	val, err := d.Config.Get("dotfile.path")
	if err != nil || val == "" {
		return "", fmt.Errorf("[ERROR] No dotfile configuration found. Please run 'dotman init' first.")
	}
	dir := val.(string)
	fs := NewFileService()
	dir = fs.ExpandHome(dir)
	stat, statErr := os.Stat(dir)
	if statErr != nil || !stat.IsDir() {
		return "", fmt.Errorf("[ERROR] Dotfile path '%s' does not exist.", dir)
	}
	return dir, nil
}

// GetHomeDir returns the absolute path to the 'home' subdirectory inside the dotman repo.
func (d *DotmanService) GetHomeDir() (string, error) {
	dir, err := d.IsInitialized()
	if err != nil {
		return "", err
	}
	homeDir := filepath.Join(dir, "home")
	return homeDir, nil
}
