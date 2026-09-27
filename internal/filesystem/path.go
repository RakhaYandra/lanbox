package filesystem

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Entry is one file or directory in a listing.
type Entry struct {
	Name string `json:"name"`
	Type string `json:"type"` // "file" or "directory"
	Size int64  `json:"size,omitempty"`
}

// Resolve maps user input to a path inside root.
// Gate: Clean -> Resolve -> Validate inside Root.
func Resolve(root, input string) (string, error) {
	decoded, err := url.PathUnescape(input)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}
	// Fail-closed: any ".." after decoding is rejected, even forms that
	// would safely clamp to root. Legit names containing ".." are refused
	// as documented tradeoff.
	if strings.Contains(decoded, "..") {
		return "", fmt.Errorf("invalid path")
	}
	cleaned := filepath.Clean("/" + decoded)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid path")
	}
	abs := filepath.Join(root, cleaned)
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid path")
	}
	if target, err := filepath.EvalSymlinks(abs); err == nil {
		if rel2, err := filepath.Rel(root, target); err != nil || rel2 == ".." ||
			strings.HasPrefix(rel2, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid path")
		}
		return target, nil
	}
	return abs, nil
}

// Browse lists entries of a resolved directory.
func Browse(dir string) ([]Entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(items))
	for _, it := range items {
		e := Entry{Name: it.Name()}
		if it.IsDir() {
			e.Type = "directory"
		} else {
			e.Type = "file"
			if info, err := it.Info(); err == nil {
				e.Size = info.Size()
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}
