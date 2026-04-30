package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func buildRequestFromDescriptorSet(t *testing.T, filesToGenerate []string, srcByName map[string]string) *pluginpb.CodeGeneratorRequest {
	t.Helper()

	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not on PATH")
	}

	tempDir := t.TempDir()
	for name, src := range srcByName {
		path := tempDir + "/" + name
		if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{
		"--include_imports",
		"--descriptor_set_out=/dev/stdout",
		"-I", tempDir,
	}
	for name := range srcByName {
		args = append(args, tempDir+"/"+name)
	}
	descriptorBytes, err := exec.Command("protoc", args...).Output()
	if err != nil {
		t.Fatalf("protoc invocation failed: %v", err)
	}

	descriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descriptorBytes, descriptorSet); err != nil {
		t.Fatalf("unmarshal descriptor set: %v", err)
	}

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: filesToGenerate,
		ProtoFile:      descriptorSet.GetFile(),
	}
}

func runPluginRequest(t *testing.T, request *pluginpb.CodeGeneratorRequest) *pluginpb.CodeGeneratorResponse {
	t.Helper()

	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var output bytes.Buffer
	if err := run(bytes.NewReader(requestBytes), &output); err != nil {
		t.Fatalf("run: %v", err)
	}

	response := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(output.Bytes(), response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("plugin reported error: %s", *response.Error)
	}
	return response
}

func dirOf(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return "."
}

// TestRunWithExampleProto exercises the full plugin pipeline by shelling out
// to protoc to build a descriptor set for examples/example.proto, wrapping it
// in a CodeGeneratorRequest, and asserting the response file matches the
// committed golden byte-for-byte. The test skips when protoc is unavailable.
func TestRunWithExampleProto(t *testing.T) {
	if _, err := exec.LookPath("protoc"); err != nil {
		t.Skip("protoc not on PATH")
	}

	descriptorBytes, err := exec.Command("protoc",
		"--include_imports",
		"--descriptor_set_out=/dev/stdout",
		"-I", "../../examples",
		"../../examples/example.proto",
	).Output()
	if err != nil {
		t.Fatalf("protoc invocation failed: %v", err)
	}

	descriptorSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descriptorBytes, descriptorSet); err != nil {
		t.Fatalf("unmarshal descriptor set: %v", err)
	}

	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"example.proto"},
		ProtoFile:      descriptorSet.GetFile(),
	}
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var output bytes.Buffer
	if err := run(bytes.NewReader(requestBytes), &output); err != nil {
		t.Fatalf("run: %v", err)
	}

	response := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(output.Bytes(), response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error != nil {
		t.Fatalf("plugin reported error: %s", *response.Error)
	}
	if len(response.File) != 2 {
		t.Fatalf("expected 2 generated files (wrapper + proto_core_utils), got %d", len(response.File))
	}
	if got, want := response.File[0].GetName(), "example.pb.gd"; got != want {
		t.Errorf("output filename = %q, want %q", got, want)
	}
	if got, want := response.File[1].GetName(), "proto_core_utils.gd"; got != want {
		t.Errorf("sibling filename = %q, want %q", got, want)
	}

	got := response.File[0].GetContent()
	want, err := os.ReadFile("../../examples/golden.gd")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(string(want), "\n")
		limit := len(gotLines)
		if len(wantLines) < limit {
			limit = len(wantLines)
		}
		for i := 0; i < limit; i++ {
			if gotLines[i] != wantLines[i] {
				t.Fatalf("output differs from golden at line %d:\n  got:  %q\n  want: %q", i+1, gotLines[i], wantLines[i])
			}
		}
		t.Fatalf("output differs from golden in length: got %d lines, want %d lines", len(gotLines), len(wantLines))
	}
}

// TestRunErrorOnInvalidProto confirms that the plugin reports validator
// failures through the response Error field rather than via a panic or a
// non-zero exit status.
func TestRunErrorOnInvalidProto(t *testing.T) {
	syntax := "proto2"
	name := "bad.proto"
	descriptor := &descriptorpb.FileDescriptorProto{
		Name:   &name,
		Syntax: &syntax,
	}
	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{name},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{descriptor},
	}
	requestBytes, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var output bytes.Buffer
	if err := run(bytes.NewReader(requestBytes), &output); err != nil {
		t.Fatalf("run returned error instead of populating response.Error: %v", err)
	}

	response := &pluginpb.CodeGeneratorResponse{}
	if err := proto.Unmarshal(output.Bytes(), response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Error == nil {
		t.Fatal("expected response.Error to be set for a proto2 input")
	}
}

func TestRunPreservesNestedTypeQualification(t *testing.T) {
	request := buildRequestFromDescriptorSet(t, []string{"nested.proto"}, map[string]string{
		"nested.proto": `syntax = "proto3";
message Outer { message Inner {} }
message Uses { Outer.Inner inner = 1; }`,
	})

	response := runPluginRequest(t, request)
	if len(response.File) != 2 {
		t.Fatalf("expected 2 generated files (wrapper + proto_core_utils), got %d", len(response.File))
	}
	got := response.File[0].GetContent()
	if !strings.Contains(got, "var _inner: Outer.Inner = null") {
		t.Fatalf("missing qualified field type:\n%s", got)
	}
	if !strings.Contains(got, "_inner = Outer.Inner.new()") {
		t.Fatalf("missing qualified constructor:\n%s", got)
	}
}

func TestRunGeneratesTransitivelyImportedFiles(t *testing.T) {
	request := buildRequestFromDescriptorSet(t, []string{"main.proto"}, map[string]string{
		"shared.proto": `syntax = "proto3";
message Shared {}`,
		"main.proto": `syntax = "proto3";
import "shared.proto";
message Uses { Shared shared = 1; }`,
	})

	response := runPluginRequest(t, request)
	names := make([]string, 0, len(response.File))
	for _, f := range response.File {
		names = append(names, f.GetName())
	}

	hasMain := false
	hasShared := false
	hasCore := false
	for _, name := range names {
		switch name {
		case "main.pb.gd":
			hasMain = true
		case "shared.pb.gd":
			hasShared = true
		case "proto_core_utils.gd":
			hasCore = true
		}
	}
	if !hasMain {
		t.Fatalf("missing main.pb.gd in %v", names)
	}
	if !hasShared {
		t.Fatalf("imported shared.pb.gd was not generated; got %v", names)
	}
	if !hasCore {
		t.Fatalf("missing proto_core_utils.gd in %v", names)
	}
}

func TestConvertToWrapperFilename(t *testing.T) {
	cases := map[string]string{
		"google/protobuf/timestamp.proto": "google/protobuf/timestamp.pb.gd",
		"foo/bar.proto":                   "foo/bar.pb.gd",
		"PlayerStats.proto":               "player_stats.pb.gd",
		"player_stats.proto":              "player_stats.pb.gd",
		"example.proto":                   "example.pb.gd",
	}
	for input, want := range cases {
		if got := convertToWrapperFilename(input); got != want {
			t.Errorf("convertToWrapperFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeProtoStem(t *testing.T) {
	cases := map[string]string{
		"PlayerStats":   "player_stats",
		"player_stats":  "player_stats",
		"HTTPResponse":  "http_response",
		"":              "proto",
		"123abc":        "proto_123abc",
		"foo--bar__baz": "foo_bar_baz",
	}
	for input, want := range cases {
		if got := normalizeProtoStem(input); got != want {
			t.Errorf("normalizeProtoStem(%q) = %q, want %q", input, got, want)
		}
	}
}
