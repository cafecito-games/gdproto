package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/cli"
	"github.com/cafecito-games/gdproto/internal/gdprotopb"
	"github.com/cafecito-games/gdproto/internal/generator"
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

func TestRootDebugLogLevelEmitsJSONLogs(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"--log-level", "debug", "-o", tempDir, inputPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}

	var sawComponent, sawDebug, sawInfo bool
	for _, line := range strings.Split(strings.TrimSpace(errOut.String()), "\n") {
		if line == "" || strings.HasPrefix(line, "wrote ") {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("non-JSON log line %q: %v", line, err)
		}
		if _, ok := record["component"]; ok {
			sawComponent = true
		}
		switch record["level"] {
		case "DEBUG":
			sawDebug = true
		case "INFO":
			sawInfo = true
		}
	}
	if !sawComponent {
		t.Errorf("expected component attribute in logs; got:\n%s", errOut.String())
	}
	if !sawDebug {
		t.Errorf("expected DEBUG logs at debug level; got:\n%s", errOut.String())
	}
	if !sawInfo {
		t.Errorf("expected INFO summary at debug level; got:\n%s", errOut.String())
	}
}

func TestRootDefaultLogLevelIsQuiet(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	var out, errOut bytes.Buffer
	code := cli.Execute([]string{"-o", tempDir, inputPath}, &out, &errOut)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, errOut.String())
	}
	if strings.Contains(errOut.String(), `"level":"DEBUG"`) ||
		strings.Contains(errOut.String(), `"level":"INFO"`) {
		t.Errorf("default warn level should not emit DEBUG/INFO logs; got:\n%s", errOut.String())
	}
	if !strings.Contains(errOut.String(), "wrote ") {
		t.Errorf("expected progress line; got:\n%s", errOut.String())
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
		// Each script is written with its .uid sidecar, so the progress line
		// reports twice as many files as there are scripts.
		sidecar := name + ".uid"
		content, err := os.ReadFile(filepath.Join(dir, sidecar))
		if err != nil {
			t.Errorf("missing %s: %v", sidecar, err)
			continue
		}
		if got, want := string(content), generator.SidecarSource(name); got != want {
			t.Errorf("%s: got %q, want %q", sidecar, got, want)
		}
	}
	if !strings.Contains(errOut.String(), "wrote 10 files to") {
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

func TestCLIEmitTextFormatDisabledAndEnabled(t *testing.T) {
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	defaultDir := t.TempDir()
	enabledDir := t.TempDir()
	bareDir := t.TempDir()
	disabledDir := t.TempDir()

	for _, runCase := range []struct {
		dir  string
		args []string
	}{
		{dir: defaultDir, args: []string{"-o", defaultDir, inputPath}},
		{dir: enabledDir, args: []string{"--emit_text_format=true", "-o", enabledDir, inputPath}},
		{dir: bareDir, args: []string{"--emit_text_format", "-o", bareDir, inputPath}},
		{dir: disabledDir, args: []string{"--emit_text_format=false", "-o", disabledDir, inputPath}},
	} {
		var out, errOut bytes.Buffer
		if code := cli.Execute(runCase.args, &out, &errOut); code != 0 {
			t.Fatalf("args %v exited %d: stderr=%s", runCase.args, code, errOut.String())
		}
	}

	entries, err := os.ReadDir(defaultDir)
	if err != nil {
		t.Fatalf("read defaultDir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		defaultBytes, err := os.ReadFile(filepath.Join(defaultDir, name))
		if err != nil {
			t.Fatalf("read default %s: %v", name, err)
		}
		enabledBytes, err := os.ReadFile(filepath.Join(enabledDir, name))
		if err != nil {
			t.Fatalf("read enabled %s: %v", name, err)
		}
		bareBytes, err := os.ReadFile(filepath.Join(bareDir, name))
		if err != nil {
			t.Fatalf("read bare %s: %v", name, err)
		}
		disabledBytes, err := os.ReadFile(filepath.Join(disabledDir, name))
		if err != nil {
			t.Fatalf("read disabled %s: %v", name, err)
		}

		if !bytes.Equal(defaultBytes, enabledBytes) {
			t.Errorf("%s differs between default and --emit_text_format=true", name)
		}
		if !bytes.Equal(defaultBytes, bareBytes) {
			t.Errorf("%s differs between default and bare --emit_text_format", name)
		}
		if strings.HasSuffix(name, ".uid") {
			if !bytes.Equal(defaultBytes, disabledBytes) {
				t.Errorf("sidecar %s changed with --emit_text_format=false", name)
			}
			continue
		}
		if name == "proto_core_utils.gd" || name == "ExamplePlayerStatus.pb.gd" {
			continue
		}
		disabledText := string(disabledBytes)
		for _, forbidden := range []string{"func to_text(", "func from_text(", ".to_text(", ".from_text(", "_get_enum_name_", "_parse_enum_value_"} {
			if strings.Contains(disabledText, forbidden) {
				t.Errorf("disabled %s unexpectedly contains %q", name, forbidden)
			}
		}
		for _, required := range []string{"func to_bytes(", "func from_bytes(", "func _to_string("} {
			if !strings.Contains(disabledText, required) {
				t.Errorf("disabled %s missing %q", name, required)
			}
		}
	}
}

func TestCLIEmitTextFormatRejectsInvalidValuesWithoutWritingOutput(t *testing.T) {
	inputPath := filepath.Join("..", "..", "examples", "example.proto")

	invalidArgs := [][]string{
		{"--emit_text_format=maybe"},
		{"--emit_text_format=0"},
		{"--emit_text_format=1"},
		{"--emit_text_format=TRUE"},
		{"--emit_text_format=true", "--emit_text_format=false"},
		{"--emit_text_format=false", "--emit_text_format=false"},
		{"--emit_text_format", "false"},
	}

	for _, flags := range invalidArgs {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			outDir := t.TempDir()
			sentinelPath := filepath.Join(outDir, "sentinel.txt")
			if err := os.WriteFile(sentinelPath, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}

			args := append([]string{}, flags...)
			args = append(args, "-o", outDir, inputPath)
			var out, errOut bytes.Buffer
			if code := cli.Execute(args, &out, &errOut); code == 0 {
				t.Fatalf("expected non-zero exit for %v, got 0", args)
			}

			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatalf("read outDir: %v", err)
			}
			if len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
				t.Errorf("output directory modified on error: got %d entries", len(entries))
			}
			data, err := os.ReadFile(sentinelPath)
			if err != nil || string(data) != "keep" {
				t.Errorf("sentinel altered: err=%v data=%q", err, string(data))
			}
		})
	}
}
