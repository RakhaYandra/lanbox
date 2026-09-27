package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	valid := []string{"", "/", "/docs", "/docs/a.txt", "docs/a.txt"}
	for _, in := range valid {
		if _, err := Resolve(root, in); err != nil {
			t.Errorf("Resolve(%q) = %v, want nil", in, err)
		}
	}

	invalid := []string{
		"..",
		"../",
		"/..",
		"/docs/../..",
		"..%2f..",
		"/docs/../../etc/passwd",
		"%2e%2e/%2e%2e/x",
	}
	for _, in := range invalid {
		if _, err := Resolve(root, in); err == nil {
			t.Errorf("Resolve(%q) = nil, want error", in)
		}
	}
}

func TestResolveSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("s"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "/link"); err == nil {
		t.Error("Resolve(symlink escape) = nil, want error")
	}
}

func TestBrowse(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := Browse(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("Browse = %d entries, want 2", len(entries))
	}
	for _, e := range entries {
		switch e.Name {
		case "f.txt":
			if e.Type != "file" || e.Size != 5 {
				t.Errorf("f.txt = %+v, want file size 5", e)
			}
		case "sub":
			if e.Type != "directory" {
				t.Errorf("sub = %+v, want directory", e)
			}
		default:
			t.Errorf("unexpected entry %+v", e)
		}
	}
}
