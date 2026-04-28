package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/cli"
)

func TestRootVersionFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	got := out.String()
	want := "gogdproto 0.1.0\n"
	if got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}

func TestRootHelpFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "gogdproto") {
		t.Fatalf("help output missing program name; got: %q", out.String())
	}
}

func TestRootNoArgsPrintsHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute(nil, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("expected help output, got: %q", out.String())
	}
}

func TestRootInvalidLogLevel(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--log-level", "loud"}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit, got 0; stdout=%q stderr=%q", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "log level") {
		t.Fatalf("expected error mentioning log level, got: %q", errOut.String())
	}
}

func TestRootValidLogLevel(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--log-level", "debug", "--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
}

func TestRootCompilesExampleProto(t *testing.T) {
	tempDir := t.TempDir()
	outPath := filepath.Join(tempDir, "out.gd")
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	var out, errOut bytes.Buffer
	code := cli.Execute([]string{inputPath, "-o", outPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Generated") {
		t.Errorf("expected success message, got: %q", out.String())
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	wantPrefix := "class_name Example\n\nextends RefCounted\n\nenum ProtobufError {"
	if !strings.HasPrefix(string(data), wantPrefix) {
		preview := string(data)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Errorf("output missing expected prefix; first 200 chars:\n%s", preview)
	}
}

func TestRootMissingOutputFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	inputPath := filepath.Join("..", "..", "examples", "example.proto")
	code := cli.Execute([]string{inputPath}, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit when -o omitted")
	}
}
