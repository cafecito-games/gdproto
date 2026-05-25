//go:build integration

package integration_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot returns the absolute path to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("git rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// must fatals the test if err is non-nil.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// copyFile copies src to dst, creating any missing parent directories.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// buildBinaries compiles gdproto and protoc-gen-gdscript into a tempdir and
// returns their paths. It uses the worktree's go module so binaries reflect
// the code under test.
func buildBinaries(t *testing.T) (gdprotoPath, pluginPath string) {
	t.Helper()
	dir := t.TempDir()
	gdprotoPath = filepath.Join(dir, "gdproto")
	pluginPath = filepath.Join(dir, "protoc-gen-gdscript")
	root := repoRoot(t)
	for _, b := range []struct{ out, pkg string }{
		{gdprotoPath, "./cmd/gdproto"},
		{pluginPath, "./cmd/protoc-gen-gdscript"},
	} {
		cmd := exec.Command("go", "build", "-o", b.out, b.pkg)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", b.pkg, err, out)
		}
	}
	return gdprotoPath, pluginPath
}
