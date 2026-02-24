package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRelPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		out  string
	}{
		{name: "plain", in: ".zshrc", out: ".zshrc"},
		{name: "leading_dot_slash", in: "./.zshrc", out: ".zshrc"},
		{name: "home_prefix", in: "home/.zshrc", out: ".zshrc"},
		{name: "dot_slash_and_home_prefix", in: "./home/.zshrc", out: ".zshrc"},
		{name: "nested", in: "home/.config/nvim/init.lua", out: ".config/nvim/init.lua"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeRelPath(tt.in); got != tt.out {
				t.Fatalf("normalizeRelPath(%q)=%q, want %q", tt.in, got, tt.out)
			}
		})
	}
}

func TestShortUniquePrefix(t *testing.T) {
	t.Parallel()

	// Different hashes should shorten to at least 7 chars and should still differ.
	a := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	b := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	a2, b2 := shortUniquePrefix(a, b)
	if len(a2) < 7 || len(b2) < 7 {
		t.Fatalf("expected prefixes length >= 7, got %d and %d", len(a2), len(b2))
	}
	if a2 == b2 {
		t.Fatalf("expected different prefixes, got %q and %q", a2, b2)
	}

	// Identical inputs should remain identical.
	a3, b3 := shortUniquePrefix(a, a)
	if a3 != a || b3 != a {
		t.Fatalf("expected unchanged when identical; got %q and %q", a3, b3)
	}
}

func TestFileHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.txt")
	p2 := filepath.Join(dir, "b.txt")

	if err := os.WriteFile(p1, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write p1: %v", err)
	}
	if err := os.WriteFile(p2, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write p2: %v", err)
	}

	h1, err := fileHash(p1)
	if err != nil {
		t.Fatalf("fileHash(p1): %v", err)
	}
	h2, err := fileHash(p2)
	if err != nil {
		t.Fatalf("fileHash(p2): %v", err)
	}
	if h1 != h2 {
		t.Fatalf("expected equal hashes for equal contents; got %q and %q", h1, h2)
	}

	if err := os.WriteFile(p2, []byte("hello2"), 0o644); err != nil {
		t.Fatalf("rewrite p2: %v", err)
	}
	h3, err := fileHash(p2)
	if err != nil {
		t.Fatalf("fileHash(p2) after rewrite: %v", err)
	}
	if h1 == h3 {
		t.Fatalf("expected different hashes after content change; got %q and %q", h1, h3)
	}
}
