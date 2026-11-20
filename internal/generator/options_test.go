package generator_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cafecito-games/gdproto/internal/generator"
)

func TestParsePluginParameters(t *testing.T) {
	t.Run("empty uses defaults", func(t *testing.T) {
		options, err := generator.ParsePluginParameters("")
		require.NoError(t, err)
		assert.Equal(t, generator.DefaultOptions(), options)
		assert.True(t, options.EmitTextFormat)
	})

	t.Run("explicit true", func(t *testing.T) {
		options, err := generator.ParsePluginParameters("emit_text_format=true")
		require.NoError(t, err)
		assert.True(t, options.EmitTextFormat)
	})

	t.Run("explicit false", func(t *testing.T) {
		options, err := generator.ParsePluginParameters("emit_text_format=false")
		require.NoError(t, err)
		assert.False(t, options.EmitTextFormat)
	})

	invalidCases := []struct {
		name      string
		parameter string
		wantError string
	}{
		{name: "unknown key", parameter: "unknown_option=true", wantError: "unknown plugin parameter"},
		{name: "unknown key alongside valid", parameter: "emit_text_format=false,extra=true", wantError: "unknown plugin parameter"},
		{name: "missing value separator", parameter: "emit_text_format", wantError: "expected key=value"},
		{name: "empty key", parameter: "=false", wantError: "expected key=value"},
		{name: "empty value", parameter: "emit_text_format=", wantError: `expected "true" or "false"`},
		{name: "numeric true", parameter: "emit_text_format=1", wantError: `expected "true" or "false"`},
		{name: "numeric false", parameter: "emit_text_format=0", wantError: `expected "true" or "false"`},
		{name: "uppercase true", parameter: "emit_text_format=TRUE", wantError: `expected "true" or "false"`},
		{name: "yes spelling", parameter: "emit_text_format=yes", wantError: `expected "true" or "false"`},
		{name: "whitespace around value", parameter: "emit_text_format= false", wantError: `expected "true" or "false"`},
		{name: "whitespace only", parameter: "   ", wantError: "expected key=value"},
		{name: "leading comma", parameter: ",emit_text_format=false", wantError: "empty parameter entry"},
		{name: "trailing comma", parameter: "emit_text_format=false,", wantError: "empty parameter entry"},
		{name: "duplicate identical", parameter: "emit_text_format=false,emit_text_format=false", wantError: "duplicate plugin parameter"},
		{name: "duplicate conflicting", parameter: "emit_text_format=true,emit_text_format=false", wantError: "duplicate plugin parameter"},
	}

	for _, testCase := range invalidCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := generator.ParsePluginParameters(testCase.parameter)
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantError)
		})
	}
}

func TestGenerateWithOptionsEmitTextFormat(t *testing.T) {
	const sharedSource = `syntax = "proto3";
package shared;

enum SharedKind {
  SHARED_KIND_UNSPECIFIED = 0;
  SHARED_KIND_ALPHA = 1;
}

message SharedDetail {
  string label = 1;
}
`
	const mainSource = `syntax = "proto3";
package app;

import "shared.proto";

enum TopState {
  TOP_STATE_UNSPECIFIED = 0;
  TOP_STATE_READY = 1;
}

message Empty {}

message Envelope {
  option (gdproto.resource) = true;

  enum LocalMode {
    LOCAL_MODE_UNSPECIFIED = 0;
    LOCAL_MODE_FAST = 1;
  }

  message Nested {
    int32 weight = 1;
    shared.SharedDetail imported = 2;
  }

  string name = 1;
  optional int32 level = 2;
  TopState state = 3;
  LocalMode mode = 4;
  shared.SharedKind kind = 5;
  shared.SharedDetail detail = 6;
  Nested nested = 7;
  repeated Nested children = 8;
  map<string, int32> scores = 9;
  map<string, Nested> by_key = 10;

  oneof choice {
    int32 number = 11;
    string text = 12;
    Nested nested_choice = 13;
    TopState state_choice = 14;
  }
}
`
	sharedFile := parseProtoSource(t, sharedSource, "shared.proto")
	mainFile := parseProtoSource(t, mainSource, "app.proto")
	imports := []generator.FileEntry{{File: sharedFile, Filename: "shared.proto"}}

	defaultFiles, err := generator.Generate(mainFile, "app.proto", imports)
	require.NoError(t, err)

	enabledFiles, err := generator.GenerateWithOptions(mainFile, "app.proto", imports, generator.Options{
		EmitTextFormat: true,
	})
	require.NoError(t, err)

	disabledFiles, err := generator.GenerateWithOptions(mainFile, "app.proto", imports, generator.Options{
		EmitTextFormat: false,
	})
	require.NoError(t, err)

	disabledFilesAgain, err := generator.GenerateWithOptions(mainFile, "app.proto", imports, generator.Options{
		EmitTextFormat: false,
	})
	require.NoError(t, err)

	require.Equal(t, len(defaultFiles), len(enabledFiles))
	require.Equal(t, len(defaultFiles), len(disabledFiles))
	require.Equal(t, len(disabledFiles), len(disabledFilesAgain))

	for i := range defaultFiles {
		assert.Equal(t, defaultFiles[i].Filename, enabledFiles[i].Filename)
		assert.Equal(t, defaultFiles[i].ClassName, enabledFiles[i].ClassName)
		assert.Equal(t, defaultFiles[i].Filename, disabledFiles[i].Filename)
		assert.Equal(t, defaultFiles[i].ClassName, disabledFiles[i].ClassName)
		assert.Equal(t, generator.SidecarSource(defaultFiles[i].Filename), generator.SidecarSource(disabledFiles[i].Filename))

		defaultSource := mustSource(t, defaultFiles[i])
		enabledSource := mustSource(t, enabledFiles[i])
		assert.Equal(t, defaultSource, enabledSource, "explicit emit_text_format=true must match default for %s", defaultFiles[i].Filename)

		disabledSource := mustSource(t, disabledFiles[i])
		disabledSourceAgain := mustSource(t, disabledFilesAgain[i])
		assert.Equal(t, disabledSource, disabledSourceAgain, "disabled generation must be deterministic for %s", disabledFiles[i].Filename)

		assert.NotContains(t, disabledSource, "func to_text(")
		assert.NotContains(t, disabledSource, "func from_text(")
		assert.NotContains(t, disabledSource, ".to_text(")
		assert.NotContains(t, disabledSource, ".from_text(")
		assert.NotContains(t, disabledSource, "_get_enum_name_")
		assert.NotContains(t, disabledSource, "_parse_enum_value_")

		if disabledFiles[i].ClassName == "AppTopState" {
			continue
		}
		assert.Contains(t, defaultSource, "func to_text(")
		assert.Contains(t, defaultSource, "func from_text(")
		assert.Contains(t, disabledSource, "func to_bytes(")
		assert.Contains(t, disabledSource, "func from_bytes(")
		assert.Contains(t, disabledSource, "func _to_string(")
	}

	envelope := findFile(disabledFiles, "AppEnvelope")
	require.NotNil(t, envelope)
	envelopeSource := mustSource(t, *envelope)
	assert.Contains(t, envelopeSource, "extends Resource")
	assert.Contains(t, envelopeSource, "@export_storage var _name: String")
	assert.Contains(t, envelopeSource, "func set_name_pb(")
	assert.Contains(t, envelopeSource, "func has_level() -> bool:")
	assert.Contains(t, envelopeSource, "func _select_choice(next_case: ChoiceOneOf) -> void:")
	assert.Contains(t, envelopeSource, "func get_choice_case() -> ChoiceOneOf:")
	assert.NotContains(t, envelopeSource, "# Enum name lookup helpers")

	empty := findFile(disabledFiles, "AppEmpty")
	require.NotNil(t, empty)
	emptySource := mustSource(t, *empty)
	assert.Contains(t, emptySource, "func to_bytes() -> PackedByteArray:")
	assert.Contains(t, emptySource, "func from_bytes(_data: PackedByteArray) -> ProtoCoreUtils.ProtobufError:")
	assert.Contains(t, emptySource, "func _to_string() -> String:")
	assert.NotContains(t, emptySource, "to_text")
	assert.NotContains(t, emptySource, "from_text")
}

func TestGenerateExampleDisabledProfileOmitsOnlyTextFormat(t *testing.T) {
	const source = `syntax = "proto3";
message Ping {
  string body = 1;
}
`
	file := parseProtoSource(t, source, "ping.proto")
	files, err := generator.GenerateWithOptions(file, "ping.proto", nil, generator.Options{EmitTextFormat: false})
	require.NoError(t, err)
	require.Len(t, files, 1)
	rendered := mustSource(t, files[0])
	assert.False(t, strings.Contains(rendered, "to_text"))
	assert.False(t, strings.Contains(rendered, "from_text"))
	assert.Contains(t, rendered, `return "Ping { " + ", ".join(parts) + " }"`)
}
