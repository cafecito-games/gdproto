//go:build integration

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIWithOptionsProto(t *testing.T) {
	gdproto, _ := buildBinaries(t)
	root := repoRoot(t)
	work := t.TempDir()
	incRoot := filepath.Join(work, "proto-include")
	must(t, copyFile(
		filepath.Join(root, "proto/gdproto/options.proto"),
		filepath.Join(incRoot, "gdproto/options.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/options/sample.proto"),
		filepath.Join(work, "sample.proto"),
	))
	outDir := filepath.Join(work, "out")
	cmd := exec.Command(gdproto, "-I", incRoot, "-o", outDir, filepath.Join(work, "sample.proto"))
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gdproto: %v", err)
	}
	assertGameHeroGenerated(t, outDir)
}

func TestProtocWithOptionsProto(t *testing.T) {
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
		filepath.Join(root, "tests/integration/fixtures/options/sample.proto"),
		filepath.Join(work, "sample.proto"),
	))
	outDir := filepath.Join(work, "out")
	must(t, os.MkdirAll(outDir, 0o755))
	cmd := exec.Command("protoc",
		"--plugin=protoc-gen-gdscript="+plugin,
		"-I", work,
		"--gdscript_out", outDir,
		"sample.proto",
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("protoc: %v", err)
	}
	assertGameHeroGenerated(t, outDir)
}

func TestBufWithOptionsProto(t *testing.T) {
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skip("buf not installed; skipping")
	}
	_, plugin := buildBinaries(t)
	root := repoRoot(t)
	work := t.TempDir()
	must(t, copyFile(
		filepath.Join(root, "proto/gdproto/options.proto"),
		filepath.Join(work, "gdproto/options.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/options/sample.proto"),
		filepath.Join(work, "sample.proto"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/options/buf.yaml"),
		filepath.Join(work, "buf.yaml"),
	))
	must(t, copyFile(
		filepath.Join(root, "tests/integration/fixtures/options/buf.gen.yaml"),
		filepath.Join(work, "buf.gen.yaml"),
	))
	cmd := exec.Command("buf", "generate")
	cmd.Dir = work
	cmd.Env = append(os.Environ(),
		"PATH="+filepath.Dir(plugin)+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("buf: %v", err)
	}
	assertGameHeroGenerated(t, filepath.Join(work, "out"))
}

func assertGameHeroGenerated(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "GameHero.pb.gd"))
	if err != nil {
		t.Fatalf("read GameHero.pb.gd: %v", err)
	}
	if !strings.Contains(string(data), "class_name GameHero") {
		t.Fatalf("missing class_name directive in GameHero.pb.gd. Content:\n%s", string(data))
	}
	if _, err := os.Stat(filepath.Join(dir, "proto_core_utils.gd")); err != nil {
		t.Fatalf("missing proto_core_utils.gd: %v", err)
	}
}
