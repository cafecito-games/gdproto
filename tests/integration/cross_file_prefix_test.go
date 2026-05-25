//go:build integration

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLIWithCrossFileClassPrefix verifies that when proto A imports proto B
// and B sets option (gdproto.class_prefix), A's generated GDScript references
// B's messages with the imported file's prefix — not a filename-derived one.
func TestCLIWithCrossFileClassPrefix(t *testing.T) {
	gdproto, _ := buildBinaries(t)
	root := repoRoot(t)
	work := t.TempDir()
	incRoot := filepath.Join(work, "proto-include")
	must(t, copyFile(
		filepath.Join(root, "proto/gdproto/options.proto"),
		filepath.Join(incRoot, "gdproto/options.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/cross_file_prefix/shared.proto"),
		filepath.Join(work, "shared.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/cross_file_prefix/main.proto"),
		filepath.Join(work, "main.proto"),
	))
	outDir := filepath.Join(work, "out")
	cmd := exec.Command(gdproto, "-I", incRoot, "-I", work, "-o", outDir, filepath.Join(work, "main.proto"))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gdproto: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "MainUse.pb.gd"))
	if err != nil {
		t.Fatalf("read MainUse.pb.gd: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "CustomFoo") {
		t.Fatalf("MainUse.pb.gd should reference CustomFoo (imported file's class_prefix); got:\n%s", got)
	}
	if strings.Contains(got, "SharedFoo") {
		t.Fatalf("MainUse.pb.gd should NOT use filename-derived SharedFoo when imported file sets class_prefix; got:\n%s", got)
	}
}

// TestProtocWithCrossFileClassPrefix exercises the plugin path through protoc.
func TestProtocWithCrossFileClassPrefix(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not installed; skipping")
	}
	_, plugin := buildBinaries(t)
	root := repoRoot(t)
	work := t.TempDir()
	must(t, copyFile(
		filepath.Join(root, "proto/gdproto/options.proto"),
		filepath.Join(work, "gdproto/options.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/cross_file_prefix/shared.proto"),
		filepath.Join(work, "shared.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/cross_file_prefix/main.proto"),
		filepath.Join(work, "main.proto"),
	))
	outDir := filepath.Join(work, "out")
	must(t, os.MkdirAll(outDir, 0o755))
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-gdscript="+plugin,
		"-I", work,
		"--gdscript_out", outDir,
		"main.proto",
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("protoc: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "MainUse.pb.gd"))
	if err != nil {
		t.Fatalf("read MainUse.pb.gd: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "CustomFoo") {
		t.Fatalf("MainUse.pb.gd should reference CustomFoo; got:\n%s", got)
	}
	if strings.Contains(got, "SharedFoo") {
		t.Fatalf("MainUse.pb.gd should NOT use filename-derived SharedFoo; got:\n%s", got)
	}
}
