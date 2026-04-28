package importer

import (
	"fmt"
	"os"
	"path/filepath"
)

// FS abstracts file lookup for import resolution.
type FS interface {
	Read(path string) ([]byte, error)
	Exists(path string) bool
}

// OSFS reads files from the real filesystem. Tries the import path
// relative to BaseDir first, then walks up parent directories looking
// for the file (matching the Python CLI's lookup strategies).
type OSFS struct {
	BaseDir string
}

// Read locates the file using the resolution strategies and returns
// its contents.
func (f *OSFS) Read(path string) ([]byte, error) {
	if found, ok := f.locate(path); ok {
		return os.ReadFile(found) //nolint:gosec // resolved path comes from import statement; OSFS users vet inputs.
	}
	return nil, fmt.Errorf("import %q not found from %s", path, f.BaseDir)
}

// Exists reports whether the import path can be located.
func (f *OSFS) Exists(path string) bool {
	_, ok := f.locate(path)
	return ok
}

// locate applies three strategies in order:
//  1. BaseDir + path.
//  2. Walk up parent directories, trying each as the prefix.
//  3. Walk up parent directories, trying just the basename.
func (f *OSFS) locate(path string) (string, bool) {
	candidate := filepath.Join(f.BaseDir, path)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, true
	}
	dir := f.BaseDir
	for {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		if c := filepath.Join(dir, path); statOK(c) {
			return c, true
		}
		if c := filepath.Join(dir, filepath.Base(path)); statOK(c) {
			return c, true
		}
	}
	return "", false
}

func statOK(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
