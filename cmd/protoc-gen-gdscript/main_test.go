package main

import (
	"bytes"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/gdproto/internal/gdprotopb"
	"github.com/cafecito-games/gdproto/internal/generator"
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

// runPluginRequestRaw is like runPluginRequest but does not fail when
// response.Error is set; callers can inspect the error directly.
func runPluginRequestRaw(t *testing.T, request *pluginpb.CodeGeneratorRequest) *pluginpb.CodeGeneratorResponse {
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
	return response
}

func dirOf(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return "."
}

func responseFilenames(response *pluginpb.CodeGeneratorResponse) []string {
	names := make([]string, 0, len(response.File))
	for _, f := range response.File {
		names = append(names, f.GetName())
	}
	sort.Strings(names)
	return names
}

// TestRunWithExampleProto exercises the full plugin pipeline by shelling out
// to protoc to build a descriptor set for examples/example.proto, wrapping
// it in a CodeGeneratorRequest, and asserting the per-class files match the
// committed goldens byte-for-byte. The test skips when protoc is unavailable.
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

	// Every emitted script is accompanied by its .uid sidecar, so that a
	// consumer's project has a stable identity for it without importing it in
	// the Godot editor first.
	wantFilenames := []string{
		"ExampleGameState.pb.gd",
		"ExampleGameState.pb.gd.uid",
		"ExamplePlayer.pb.gd",
		"ExamplePlayer.pb.gd.uid",
		"ExamplePlayerPosition.pb.gd",
		"ExamplePlayerPosition.pb.gd.uid",
		"ExamplePlayerStatus.pb.gd",
		"ExamplePlayerStatus.pb.gd.uid",
		"proto_core_utils.gd",
		"proto_core_utils.gd.uid",
	}
	gotNames := responseFilenames(response)
	if !equalStringSlices(gotNames, wantFilenames) {
		t.Fatalf("emitted filenames mismatch:\n  got:  %v\n  want: %v", gotNames, wantFilenames)
	}

	contents := map[string]string{}
	for _, f := range response.File {
		contents[f.GetName()] = f.GetContent()
	}

	for _, filename := range []string{
		"ExampleGameState.pb.gd",
		"ExamplePlayer.pb.gd",
		"ExamplePlayerPosition.pb.gd",
		"ExamplePlayerStatus.pb.gd",
	} {
		want, err := os.ReadFile("../../examples/golden/" + filename)
		if err != nil {
			t.Fatalf("read golden %s: %v", filename, err)
		}
		got := contents[filename]
		if got != string(want) {
			gotLines := strings.Split(got, "\n")
			wantLines := strings.Split(string(want), "\n")
			limit := len(gotLines)
			if len(wantLines) < limit {
				limit = len(wantLines)
			}
			for i := 0; i < limit; i++ {
				if gotLines[i] != wantLines[i] {
					t.Fatalf("%s differs from golden at line %d:\n  got:  %q\n  want: %q", filename, i+1, gotLines[i], wantLines[i])
				}
			}
			t.Fatalf("%s differs from golden in length: got %d lines, want %d lines", filename, len(gotLines), len(wantLines))
		}
	}

	for _, filename := range wantFilenames {
		if !strings.HasSuffix(filename, ".uid") {
			continue
		}
		script := strings.TrimSuffix(filename, ".uid")
		if got, want := contents[filename], generator.SidecarSource(script); got != want {
			t.Errorf("%s: got %q, want %q", filename, got, want)
		}
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
	// Per-class output: NestedOuter.pb.gd, NestedOuterInner.pb.gd,
	// NestedUses.pb.gd, plus proto_core_utils.gd, each with a .uid sidecar.
	if got, want := len(response.File), 8; got != want {
		t.Fatalf("expected %d generated files, got %d (%v)", want, got, responseFilenames(response))
	}
	contents := map[string]string{}
	for _, f := range response.File {
		contents[f.GetName()] = f.GetContent()
	}
	uses, ok := contents["NestedUses.pb.gd"]
	if !ok {
		t.Fatalf("missing NestedUses.pb.gd in %v", responseFilenames(response))
	}
	if !strings.Contains(uses, "var _inner: NestedOuterInner = null") {
		t.Fatalf("missing flattened field type:\n%s", uses)
	}
	if !strings.Contains(uses, "_inner = NestedOuterInner.new()") {
		t.Fatalf("missing flattened constructor:\n%s", uses)
	}
}

// TestRunOnlyGeneratesRequestedFiles confirms the plugin honors the standard
// protoc contract: only files listed in file_to_generate are emitted, even
// when the request supplies descriptors for transitively-imported files.
// Users who want wrappers for imported messages must list those .proto files
// explicitly in the generator input set.
func TestRunOnlyGeneratesRequestedFiles(t *testing.T) {
	request := buildRequestFromDescriptorSet(t, []string{"main.proto"}, map[string]string{
		"shared.proto": `syntax = "proto3";
message Shared {}`,
		"main.proto": `syntax = "proto3";
import "shared.proto";
message Uses { Shared shared = 1; }`,
	})

	response := runPluginRequest(t, request)
	names := responseFilenames(response)

	wantPresent := []string{"MainUses.pb.gd", "proto_core_utils.gd"}
	for _, name := range wantPresent {
		found := false
		for _, got := range names {
			if got == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %s in %v", name, names)
		}
	}
	for _, got := range names {
		if got == "SharedShared.pb.gd" {
			t.Fatalf("SharedShared.pb.gd should not be emitted when shared.proto is not in file_to_generate; got %v", names)
		}
	}
}

// TestRunGeneratesAllRequestedFiles confirms that listing both a file and
// its imports in file_to_generate emits wrappers for both and that cross-file
// type references render with the imported file's class prefix.
func TestRunGeneratesAllRequestedFiles(t *testing.T) {
	request := buildRequestFromDescriptorSet(t, []string{"main.proto", "shared.proto"}, map[string]string{
		"shared.proto": `syntax = "proto3";
message Shared {}`,
		"main.proto": `syntax = "proto3";
import "shared.proto";
message Uses { Shared shared = 1; }`,
	})

	response := runPluginRequest(t, request)
	names := responseFilenames(response)

	for _, want := range []string{"MainUses.pb.gd", "SharedShared.pb.gd", "proto_core_utils.gd"} {
		found := false
		for _, got := range names {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %s in %v", want, names)
		}
	}

	var usesSource string
	for _, f := range response.File {
		if f.GetName() == "MainUses.pb.gd" {
			usesSource = f.GetContent()
			break
		}
	}
	if !strings.Contains(usesSource, "SharedShared") {
		t.Fatalf("MainUses.pb.gd should reference SharedShared for the cross-file field type:\n%s", usesSource)
	}
}

func TestRunReportsClassNameCollision(t *testing.T) {
	// foo_bar.proto derives prefix "FooBar" and produces FooBarBaz.pb.gd.
	// foo.proto derives prefix "Foo" and produces FooBarBaz.pb.gd from its
	// "BarBaz" message. Both files target the same on-disk filename without
	// a symbol clash inside protoc.
	request := buildRequestFromDescriptorSet(t, []string{"foo_bar.proto", "foo.proto"}, map[string]string{
		"foo_bar.proto": `syntax = "proto3";
message Baz {}`,
		"foo.proto": `syntax = "proto3";
message BarBaz {}`,
	})

	response := runPluginRequestRaw(t, request)
	if response.Error == nil {
		t.Fatalf("expected collision error, got files %v", responseFilenames(response))
	}
	message := *response.Error
	if !strings.Contains(message, "class name collision") || !strings.Contains(message, `"FooBarBaz"`) {
		t.Errorf("error message missing collision detail: %q", message)
	}
	if !strings.Contains(message, "foo.proto") || !strings.Contains(message, "foo_bar.proto") {
		t.Errorf("error message missing source filenames: %q", message)
	}
	if !strings.Contains(message, "set option (gdproto.class_prefix)") {
		t.Errorf("error message missing class_prefix hint: %q", message)
	}
}

func TestPrintOptionsProtoMatchesEmbed(t *testing.T) {
	var buf bytes.Buffer
	n, err := printOptionsProto(&buf)
	if err != nil {
		t.Fatalf("printOptionsProto: %v", err)
	}
	if n != len(gdprotopb.Bytes()) {
		t.Fatalf("wrote %d bytes, want %d", n, len(gdprotopb.Bytes()))
	}
	if !bytes.Equal(buf.Bytes(), gdprotopb.Bytes()) {
		t.Fatalf("printed bytes differ from gdprotopb.Bytes()")
	}
}

func samplePluginDescriptorRequest(parameter *string) *pluginpb.CodeGeneratorRequest {
	syntax := "proto3"
	filename := "sample.proto"
	enumName := "State"
	enumZero := "STATE_UNSPECIFIED"
	enumOne := "STATE_ACTIVE"
	messageName := "Packet"
	nestedName := "Inner"
	textFieldName := "label"
	enumFieldName := "state"
	nestedFieldName := "inner"
	innerValueName := "value"
	oneofName := "choice"
	oneofIntName := "number"
	oneofStrName := "note"
	oneofIndex := int32(0)

	return &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{filename},
		Parameter:      parameter,
		ProtoFile: []*descriptorpb.FileDescriptorProto{{
			Name:   &filename,
			Syntax: &syntax,
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: &enumName,
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: &enumZero, Number: proto.Int32(0)},
					{Name: &enumOne, Number: proto.Int32(1)},
				},
			}},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: &messageName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   &textFieldName,
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     &enumFieldName,
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".State"),
					},
					{
						Name:     &nestedFieldName,
						Number:   proto.Int32(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".Packet.Inner"),
					},
					{
						Name:       &oneofIntName,
						Number:     proto.Int32(4),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
						OneofIndex: &oneofIndex,
					},
					{
						Name:       &oneofStrName,
						Number:     proto.Int32(5),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						OneofIndex: &oneofIndex,
					},
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: &oneofName}},
				NestedType: []*descriptorpb.DescriptorProto{{
					Name: &nestedName,
					Field: []*descriptorpb.FieldDescriptorProto{{
						Name:   &innerValueName,
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					}},
				}},
			}},
		}},
	}
}

func TestRunParameterEmitTextFormat(t *testing.T) {
	defaultResponse := runPluginRequest(t, samplePluginDescriptorRequest(nil))
	enabledResponse := runPluginRequest(t, samplePluginDescriptorRequest(proto.String("emit_text_format=true")))
	disabledResponse := runPluginRequest(t, samplePluginDescriptorRequest(proto.String("emit_text_format=false")))

	if !equalStringSlices(responseFilenames(defaultResponse), responseFilenames(enabledResponse)) {
		t.Fatalf("enabled filenames mismatch: got %v, want %v", responseFilenames(enabledResponse), responseFilenames(defaultResponse))
	}
	if !equalStringSlices(responseFilenames(defaultResponse), responseFilenames(disabledResponse)) {
		t.Fatalf("disabled filenames mismatch: got %v, want %v", responseFilenames(disabledResponse), responseFilenames(defaultResponse))
	}

	defaultByName := map[string]string{}
	for _, f := range defaultResponse.File {
		defaultByName[f.GetName()] = f.GetContent()
	}
	enabledByName := map[string]string{}
	for _, f := range enabledResponse.File {
		enabledByName[f.GetName()] = f.GetContent()
	}
	disabledByName := map[string]string{}
	for _, f := range disabledResponse.File {
		disabledByName[f.GetName()] = f.GetContent()
	}

	for name, defaultContent := range defaultByName {
		if enabledByName[name] != defaultContent {
			t.Errorf("%s differs between omitted parameter and emit_text_format=true", name)
		}
		if strings.HasSuffix(name, ".uid") {
			if disabledByName[name] != defaultContent {
				t.Errorf("sidecar %s changed when emit_text_format=false", name)
			}
			continue
		}
		if name == "proto_core_utils.gd" || name == "SampleState.pb.gd" {
			continue
		}
		disabledContent := disabledByName[name]
		if !strings.Contains(defaultContent, "func to_text(") || !strings.Contains(defaultContent, "func from_text(") {
			t.Errorf("default %s should contain to_text and from_text", name)
		}
		for _, forbidden := range []string{"func to_text(", "func from_text(", ".to_text(", ".from_text(", "_get_enum_name_", "_parse_enum_value_"} {
			if strings.Contains(disabledContent, forbidden) {
				t.Errorf("disabled %s unexpectedly contains %q", name, forbidden)
			}
		}
		for _, required := range []string{"func to_bytes(", "func from_bytes(", "func _to_string("} {
			if !strings.Contains(disabledContent, required) {
				t.Errorf("disabled %s missing required method %q", name, required)
			}
		}
	}
}

func TestRunRejectsInvalidPluginParametersWithoutEmittingFiles(t *testing.T) {
	cases := []struct {
		name      string
		parameter string
		wantError string
	}{
		{name: "unknown key", parameter: "bogus=true", wantError: "unknown plugin parameter"},
		{name: "invalid bool", parameter: "emit_text_format=yes", wantError: `expected "true" or "false"`},
		{name: "numeric bool", parameter: "emit_text_format=0", wantError: `expected "true" or "false"`},
		{name: "bare key", parameter: "emit_text_format", wantError: "expected key=value"},
		{name: "duplicate conflicting", parameter: "emit_text_format=true,emit_text_format=false", wantError: "duplicate plugin parameter"},
		{name: "empty entry", parameter: "emit_text_format=false,", wantError: "empty parameter entry"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			response := runPluginRequestRaw(t, samplePluginDescriptorRequest(proto.String(testCase.parameter)))
			if response.Error == nil {
				t.Fatalf("expected response.Error for parameter %q, got files %v", testCase.parameter, responseFilenames(response))
			}
			if !strings.Contains(*response.Error, testCase.wantError) {
				t.Errorf("response.Error = %q, want substring %q", *response.Error, testCase.wantError)
			}
			if len(response.File) != 0 {
				t.Errorf("expected no emitted files on parameter error, got %v", responseFilenames(response))
			}
		})
	}
}

func TestRunEmitsNoPartialFilesOnMidRequestError(t *testing.T) {
	goodSyntax := "proto3"
	goodName := "good.proto"
	goodMessage := "Ok"
	badSyntax := "proto2"
	badName := "bad.proto"

	request := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{goodName, badName},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:   &goodName,
				Syntax: &goodSyntax,
				MessageType: []*descriptorpb.DescriptorProto{{
					Name: &goodMessage,
				}},
			},
			{
				Name:   &badName,
				Syntax: &badSyntax,
			},
		},
	}

	response := runPluginRequestRaw(t, request)
	if response.Error == nil {
		t.Fatal("expected response.Error when second file fails validation")
	}
	if len(response.File) != 0 {
		t.Errorf("expected zero partial files on failure, got %v", responseFilenames(response))
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
