//go:build integration

package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmitTextFormatDisabledParityAcrossCLIProtocAndBuf(t *testing.T) {
	gdproto, plugin := buildBinaries(t)
	root := repoRoot(t)
	work := t.TempDir()

	must(t, copyFile(
		filepath.Join(root, "examples/example.proto"),
		filepath.Join(work, "example.proto"),
	))
	must(t, os.WriteFile(filepath.Join(work, "buf.yaml"), []byte("version: v2\nmodules:\n  - path: .\n"), 0o600))
	must(t, os.WriteFile(filepath.Join(work, "buf.gen.yaml"), []byte(
		"version: v2\nplugins:\n  - local: protoc-gen-gdscript\n    out: out-buf\n    opt:\n      - emit_text_format=false\n",
	), 0o600))

	cliOut := filepath.Join(work, "out-cli")
	cliCmd := exec.Command(gdproto, "--emit_text_format=false", "-o", cliOut, filepath.Join(work, "example.proto"))
	if output, err := cliCmd.CombinedOutput(); err != nil {
		t.Fatalf("gdproto failed: %v\n%s", err, output)
	}

	assertDisabledOutputValid(t, cliOut)

	if _, err := exec.LookPath("buf"); err == nil {
		bufOut := filepath.Join(work, "out-buf")
		bufCmd := exec.Command("buf", "generate")
		bufCmd.Dir = work
		bufCmd.Env = append(os.Environ(),
			"PATH="+filepath.Dir(plugin)+string(os.PathListSeparator)+os.Getenv("PATH"),
		)
		if output, err := bufCmd.CombinedOutput(); err != nil {
			t.Fatalf("buf generate failed: %v\n%s", err, output)
		}
		assertDirectoriesByteIdentical(t, cliOut, bufOut)
	}

	if _, err := exec.LookPath("protoc"); err == nil {
		protocOut := filepath.Join(work, "out-protoc")
		must(t, os.MkdirAll(protocOut, 0o755))
		protocCmd := exec.Command("protoc",
			"--plugin=protoc-gen-gdscript="+plugin,
			"-I", work,
			"--gdscript_out="+protocOut,
			"--gdscript_opt=emit_text_format=false",
			"example.proto",
		)
		if output, err := protocCmd.CombinedOutput(); err != nil {
			t.Fatalf("protoc failed: %v\n%s", err, output)
		}
		assertDirectoriesByteIdentical(t, cliOut, protocOut)
	}
}

func TestEmitTextFormatInvalidOptionFailsClosedAcrossEntryPoints(t *testing.T) {
	gdproto, plugin := buildBinaries(t)
	root := repoRoot(t)
	work := t.TempDir()

	must(t, copyFile(
		filepath.Join(root, "examples/example.proto"),
		filepath.Join(work, "example.proto"),
	))

	t.Run("cli invalid value leaves output directory untouched", func(t *testing.T) {
		outDir := filepath.Join(work, "cli-invalid")
		must(t, os.MkdirAll(outDir, 0o755))
		cmd := exec.Command(gdproto, "--emit_text_format=nope", "-o", outDir, filepath.Join(work, "example.proto"))
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected error for invalid CLI flag, output:\n%s", output)
		}
		entries, readErr := os.ReadDir(outDir)
		must(t, readErr)
		if len(entries) != 0 {
			t.Fatalf("expected empty output dir after CLI error, got %d files", len(entries))
		}
	})

	t.Run("buf invalid opt writes no files", func(t *testing.T) {
		if _, err := exec.LookPath("buf"); err != nil {
			t.Skip("buf not installed; skipping")
		}
		bufDir := filepath.Join(work, "buf-invalid")
		must(t, copyFile(
			filepath.Join(root, "examples/example.proto"),
			filepath.Join(bufDir, "example.proto"),
		))
		must(t, os.WriteFile(filepath.Join(bufDir, "buf.yaml"), []byte("version: v2\nmodules:\n  - path: .\n"), 0o600))
		must(t, os.WriteFile(filepath.Join(bufDir, "buf.gen.yaml"), []byte(
			"version: v2\nplugins:\n  - local: protoc-gen-gdscript\n    out: out\n    opt:\n      - unknown_key=true\n",
		), 0o600))
		cmd := exec.Command("buf", "generate")
		cmd.Dir = bufDir
		cmd.Env = append(os.Environ(),
			"PATH="+filepath.Dir(plugin)+string(os.PathListSeparator)+os.Getenv("PATH"),
		)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected buf generate error for unknown opt, output:\n%s", output)
		}
		if !strings.Contains(string(output), "unknown plugin parameter") {
			t.Errorf("expected actionable error in buf output, got:\n%s", output)
		}
		if _, statErr := os.Stat(filepath.Join(bufDir, "out")); statErr == nil {
			entries, readErr := os.ReadDir(filepath.Join(bufDir, "out"))
			must(t, readErr)
			if len(entries) != 0 {
				t.Fatalf("expected no generated files after buf error, got %d", len(entries))
			}
		}
	})

	t.Run("protoc invalid opt writes no files", func(t *testing.T) {
		if _, err := exec.LookPath("protoc"); err != nil {
			t.Skip("protoc not installed; skipping")
		}
		outDir := filepath.Join(work, "protoc-invalid")
		must(t, os.MkdirAll(outDir, 0o755))
		cmd := exec.Command("protoc",
			"--plugin=protoc-gen-gdscript="+plugin,
			"-I", work,
			"--gdscript_out="+outDir,
			"--gdscript_opt=emit_text_format=true,emit_text_format=false",
			"example.proto",
		)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("expected protoc error for conflicting opt, output:\n%s", output)
		}
		if !strings.Contains(string(output), "duplicate plugin parameter") {
			t.Errorf("expected actionable error in protoc output, got:\n%s", output)
		}
		entries, readErr := os.ReadDir(outDir)
		must(t, readErr)
		if len(entries) != 0 {
			t.Fatalf("expected empty output dir after protoc error, got %d files", len(entries))
		}
	})
}

func assertDisabledOutputValid(t *testing.T, dir string) {
	t.Helper()
	for _, filename := range []string{
		"ExamplePlayer.pb.gd",
		"ExamplePlayerPosition.pb.gd",
		"ExampleGameState.pb.gd",
	} {
		data, err := os.ReadFile(filepath.Join(dir, filename))
		must(t, err)
		text := string(data)
		for _, forbidden := range []string{"func to_text(", "func from_text(", ".to_text(", ".from_text(", "_get_enum_name_", "_parse_enum_value_"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s unexpectedly contains %q", filename, forbidden)
			}
		}
		for _, required := range []string{"func to_bytes(", "func from_bytes(", "func _to_string("} {
			if !strings.Contains(text, required) {
				t.Errorf("%s missing %q", filename, required)
			}
		}
	}
}

func assertDirectoriesByteIdentical(t *testing.T, dirA, dirB string) {
	t.Helper()
	entriesA, err := os.ReadDir(dirA)
	must(t, err)
	entriesB, err := os.ReadDir(dirB)
	must(t, err)
	if len(entriesA) != len(entriesB) {
		t.Fatalf("directory file count mismatch: %s has %d, %s has %d", dirA, len(entriesA), dirB, len(entriesB))
	}
	for i := range entriesA {
		nameA := entriesA[i].Name()
		nameB := entriesB[i].Name()
		if nameA != nameB {
			t.Fatalf("filename mismatch: %s vs %s", nameA, nameB)
		}
		bytesA, err := os.ReadFile(filepath.Join(dirA, nameA))
		must(t, err)
		bytesB, err := os.ReadFile(filepath.Join(dirB, nameB))
		must(t, err)
		if !bytes.Equal(bytesA, bytesB) {
			t.Errorf("%s differs between %s and %s", nameA, dirA, dirB)
		}
	}
}
