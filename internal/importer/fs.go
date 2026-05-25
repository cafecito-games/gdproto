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

// OSFS reads files from the real filesystem. Tries each entry in
// IncludePaths first (mirroring protoc's `-I` semantics), then the
// import path relative to BaseDir, then walks up parent directories
// looking for the file.
type OSFS struct {
	BaseDir      string
	IncludePaths []string
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

// locate applies the following strategies in order:
//  1. For each IncludePaths entry, IncludePath + path.
//  2. BaseDir + path.
//  3. Walk up parent directories, trying each as the prefix.
//  4. Walk up parent directories, trying just the basename.
func (f *OSFS) locate(path string) (string, bool) {
	for _, inc := range f.IncludePaths {
		if c := filepath.Join(inc, path); statOK(c) {
			return c, true
		}
	}
	candidate := filepath.Join(f.BaseDir, path)
	if statOK(candidate) {
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
