package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileService struct{}

func NewFileService() *FileService {
	return &FileService{}
}

// ExpandHome replaces a leading "~" with the user's home directory.
// Does not attempt to resolve "~user" forms.
func (fs *FileService) ExpandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if len(path) > 1 && (path[1] == '/' || path[1] == os.PathSeparator) {
		return filepath.Join(home, path[2:])
	}
	return path
}

// HomeDir returns the current user's home directory.
func (fs *FileService) HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// IsAbs reports whether the path is absolute.
func (fs *FileService) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// Join joins any number of path elements into a single path.
func (fs *FileService) Join(paths ...string) string {
	return filepath.Join(paths...)
}

// Rel returns a relative path from base to target.
func (fs *FileService) Rel(base, target string) (string, error) {
	return filepath.Rel(base, target)
}

// Stat returns file info for the given path.
func (fs *FileService) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// MkdirAll creates a directory and all necessary parents.
func (fs *FileService) MkdirAll(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

// Exists checks if a file exists at the given path.
func (fs *FileService) Exists(path string) error {
	_, err := os.Stat(path)
	return err
}

// CopyFile copies a file from src to dst, preserving permissions.
func (fs *FileService) CopyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy file %s to %s: %w", src, dst, err)
	}
	if err = out.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}
	return nil
}
