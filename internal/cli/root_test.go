package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/cli"
	"github.com/cafecito-games/gdproto/internal/gdprotopb"
)

func TestRootVersionFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--version"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	got := out.String()
	want := "gdproto 0.1.0\n"
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
	if !strings.Contains(out.String(), "gdproto") {
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
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	var out, errOut bytes.Buffer
	code := cli.Execute([]string{inputPath, "-o", tempDir}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
	playerPath := filepath.Join(tempDir, "ExamplePlayer.pb.gd")
	data, err := os.ReadFile(playerPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), "class_name ExamplePlayer\n") {
		preview := string(data)
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Errorf("ExamplePlayer.pb.gd missing class_name; first 200 chars:\n%s", preview)
	}
	if !strings.Contains(string(data), "# Source: example.proto") {
		t.Errorf("ExamplePlayer.pb.gd missing source comment")
	}
}

func TestRootCreatesMissingOutputDir(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "nested", "out")
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	var out, errOut bytes.Buffer
	code := cli.Execute([]string{inputPath, "-o", outDir}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "proto_core_utils.gd")); err != nil {
		t.Fatalf("missing proto_core_utils.gd: %v", err)
	}
}

func TestCLIWritesPerClassFilesToDirectory(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--output", dir, "../../examples/example.proto"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	must := []string{
		"ExamplePlayer.pb.gd", "ExamplePlayerPosition.pb.gd",
		"ExampleGameState.pb.gd", "ExamplePlayerStatus.pb.gd",
		"proto_core_utils.gd",
	}
	for _, name := range must {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	if !strings.Contains(errOut.String(), "wrote 5 files to") {
		t.Errorf("missing progress line in stderr: %q", errOut.String())
	}
}

func TestCLIRejectsFilePathAsOutput(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--output", "some/file.gd", "../../examples/example.proto"}, &out, &errOut)
	if code == 0 {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(errOut.String(), "-o must be a directory") {
		t.Fatalf("missing directory error in %q", errOut.String())
	}
}

func TestCLIPrintOptionsProto(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--print-options-proto"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	if !bytes.Equal(out.Bytes(), gdprotopb.Bytes()) {
		t.Fatalf("stdout differs from gdprotopb.Bytes() (got %d bytes, want %d)", out.Len(), len(gdprotopb.Bytes()))
	}
}

func TestCLIResolvesImportFromIncludePath(t *testing.T) {
	work := t.TempDir()
	inc := filepath.Join(work, "include")
	src := filepath.Join(work, "src")
	if err := os.MkdirAll(filepath.Join(inc, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	sharedProto := `syntax = "proto3"; package shared; message Foo { string name = 1; }`
	if err := os.WriteFile(filepath.Join(inc, "shared", "foo.proto"), []byte(sharedProto), 0o644); err != nil {
		t.Fatal(err)
	}
	appProto := `syntax = "proto3"; import "shared/foo.proto"; message Use { shared.Foo f = 1; }`
	if err := os.WriteFile(filepath.Join(src, "app.proto"), []byte(appProto), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(work, "out")
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"-I", inc, "-o", outDir, filepath.Join(src, "app.proto")}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: stderr=%s stdout=%s", code, errOut.String(), out.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "AppUse.pb.gd")); err != nil {
		t.Fatalf("missing AppUse.pb.gd: %v", err)
	}
}

func TestCLILongFormProtoPathFlag(t *testing.T) {
	work := t.TempDir()
	inc := filepath.Join(work, "include")
	src := filepath.Join(work, "src")
	if err := os.MkdirAll(filepath.Join(inc, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}

	sharedProto := `syntax = "proto3"; package shared; message Foo { string name = 1; }`
	if err := os.WriteFile(filepath.Join(inc, "shared", "foo.proto"), []byte(sharedProto), 0o644); err != nil {
		t.Fatal(err)
	}
	appProto := `syntax = "proto3"; import "shared/foo.proto"; message Use { shared.Foo f = 1; }`
	if err := os.WriteFile(filepath.Join(src, "app.proto"), []byte(appProto), 0o644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(work, "out")
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--proto_path", inc, "-o", outDir, filepath.Join(src, "app.proto")}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d: stderr=%s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "AppUse.pb.gd")); err != nil {
		t.Fatalf("missing AppUse.pb.gd: %v", err)
	}
}

func TestCLINoArgsShowsHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Execute([]string{}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "gdproto") {
		t.Errorf("help did not include 'gdproto'")
	}
}
