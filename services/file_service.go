package services

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dotman/types"
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

// FileHash returns the hex-encoded SHA-256 hash of the file at path.
func (fs *FileService) FileHash(path string) (string, error) {
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

// ShortUniquePrefix truncates two hash strings to the shortest prefix (min 7)
// that still distinguishes them.
func ShortUniquePrefix(a, b string) (string, string) {
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

// NormalizeRelPath strips leading "./" and "home/" prefixes from a relative path.
func NormalizeRelPath(rel string) string {
	rel = strings.TrimPrefix(rel, "./")
	if strings.HasPrefix(rel, "home/") {
		rel = strings.TrimPrefix(rel, "home/")
	}
	return rel
}

// CompareFiles walks repoHome and compares each file against the corresponding
// file in userHome. Returns two lists: files that differ (changed) and files
// that exist only in the repo (created).
func (fs *FileService) CompareFiles(repoHome, userHome string) (changed []types.FileDiff, created []types.FileDiff, err error) {
	err = filepath.Walk(repoHome, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relPath := NormalizeRelPath(mustRel(repoHome, path))
		userFile := filepath.Join(userHome, relPath)
		repoHash, _ := fs.FileHash(path)
		userHash := "missing"
		repoDate := "missing"
		userDate := "missing"
		if stat, err := os.Stat(path); err == nil {
			repoDate = stat.ModTime().Format("2006-01-02 15:04:05")
		}
		if stat, err := os.Stat(userFile); err == nil {
			userHash, _ = fs.FileHash(userFile)
			userDate = stat.ModTime().Format("2006-01-02 15:04:05")
		}
		if repoHash != "missing" && userHash != "missing" {
			repoHash, userHash = ShortUniquePrefix(repoHash, userHash)
		}
		if userHash == "missing" {
			created = append(created, types.FileDiff{
				RelPath:  relPath,
				RepoHash: repoHash,
				UserHash: userHash,
				RepoDate: repoDate,
				UserDate: userDate,
			})
		} else if repoHash != userHash {
			changed = append(changed, types.FileDiff{
				RelPath:  relPath,
				RepoHash: repoHash,
				UserHash: userHash,
				RepoDate: repoDate,
				UserDate: userDate,
			})
		}
		return nil
	})
	return
}

func mustRel(base, target string) string {
	rel, _ := filepath.Rel(base, target)
	return rel
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
