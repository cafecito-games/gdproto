package generator_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/generator"
	"github.com/cafecito-games/gdproto/internal/lexer"
	"github.com/cafecito-games/gdproto/internal/parser"
	"github.com/cafecito-games/gdproto/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findFile returns the GeneratedFile in files whose ClassName matches name,
// or nil if no such file was produced.
func findFile(files []generator.GeneratedFile, name string) *generator.GeneratedFile {
	for i := range files {
		if files[i].ClassName == name {
			return &files[i]
		}
	}
	return nil
}

// classNames returns the sorted list of ClassName values for diagnostics.
func classNames(files []generator.GeneratedFile) []string {
	out := make([]string, 0, len(files))
	for _, f := range files {
		out = append(out, f.ClassName)
	}
	sort.Strings(out)
	return out
}

func TestGenerateEmptyProto(t *testing.T) {
	file := &ast.ProtoFile{Syntax: "proto3"}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files for empty proto, got %d: %v", len(files), classNames(files))
	}
}

func TestGenerateFieldlessMessageCollapsesMethods(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax:   "proto3",
		Messages: []*ast.Message{{Name: "LeaveParty"}},
	}
	files, err := generator.Generate(file, "party.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "PartyLeaveParty")
	if f == nil {
		t.Fatalf("missing PartyLeaveParty class; got %v", classNames(files))
	}
	out := f.Source()

	for _, want := range []string{
		"func to_bytes() -> PackedByteArray:",
		"\treturn PackedByteArray()",
		"func from_bytes(_data: PackedByteArray) -> ProtoCoreUtils.ProtobufError:",
		"func to_text(_indent_level: int = 0) -> String:",
		"\treturn \"\"",
		"func from_text(_text: String) -> ProtoCoreUtils.ProtobufError:",
		"\treturn ProtoCoreUtils.ProtobufError.NO_ERRORS",
		"return \"LeaveParty {}\"",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fieldless message output missing fragment %q\n--- full output ---\n%s", want, out)
		}
	}

	for _, unwanted := range []string{"# Fields", "# Accessors", "while ", "var result", "var parts"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("fieldless message output should not contain %q\n--- full output ---\n%s", unwanted, out)
		}
	}
}

func TestGenerateHeaderUsesBasename(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Foo",
			Fields: []*ast.Field{
				{FieldType: "string", Name: "name", Number: 1},
			},
		}},
	}
	files, err := generator.Generate(file, "/tmp/foo/bar_baz.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "TmpFooBarBazFoo")
	if f == nil {
		t.Fatalf("missing TmpFooBarBazFoo class; got %v", classNames(files))
	}
	if !strings.Contains(f.Source(), "# Source: bar_baz.proto") {
		t.Errorf("output should reference basename of input path; got:\n%s", f.Source())
	}
}

func TestGenerateTopLevelEnum(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Enums: []*ast.Enum{{
			Name: "PlayerStatus",
			Values: []*ast.EnumValue{
				{Name: "OFFLINE", Number: 0},
				{Name: "ONLINE", Number: 1},
				{Name: "AWAY", Number: 2},
				{Name: "IN_GAME", Number: 3},
			},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "ExamplePlayerStatus")
	if f == nil {
		t.Fatalf("missing ExamplePlayerStatus wrapper class; got %v", classNames(files))
	}
	out := f.Source()
	if !strings.Contains(out, "class_name ExamplePlayerStatus") {
		t.Errorf("output missing class_name directive:\n%s", out)
	}
	if !strings.Contains(out, "enum PlayerStatus {") {
		t.Errorf("output missing PlayerStatus enum:\n%s", out)
	}
	for _, want := range []string{"OFFLINE = 0", "ONLINE = 1", "AWAY = 2", "IN_GAME = 3"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing enum entry %q", want)
		}
	}
}

func TestGenerateMessageClassShell(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Player",
			Fields: []*ast.Field{
				{FieldType: "string", Name: "username", Number: 1},
				{FieldType: "int32", Name: "level", Number: 2},
				{FieldType: "string", Name: "inventory", Number: 5, Repeated: true},
			},
			NestedMessages: []*ast.Message{{
				Name: "Position",
				Fields: []*ast.Field{
					{FieldType: "float", Name: "x", Number: 1},
				},
			}},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	player := findFile(files, "ExamplePlayer")
	if player == nil {
		t.Fatalf("missing ExamplePlayer; got %v", classNames(files))
	}
	position := findFile(files, "ExamplePlayerPosition")
	if position == nil {
		t.Fatalf("missing ExamplePlayerPosition sibling; got %v", classNames(files))
	}

	playerOut := player.Source()
	for _, want := range []string{
		"class_name ExamplePlayer",
		"extends RefCounted",
		`var _username: String = ""`,
		"var _level: int = 0",
		"var _inventory: Array[String] = []",
		"func set_username(value: String) -> void:",
		"func get_username() -> String:",
		"func add_inventory(value: String) -> void:",
	} {
		if !strings.Contains(playerOut, want) {
			t.Errorf("ExamplePlayer missing fragment %q", want)
		}
	}
	if strings.Contains(playerOut, "class Position extends RefCounted") {
		t.Errorf("ExamplePlayer should not embed Position as a nested class:\n%s", playerOut)
	}

	positionOut := position.Source()
	for _, want := range []string{
		"class_name ExamplePlayerPosition",
		"extends RefCounted",
		"var _x: float = 0.0",
	} {
		if !strings.Contains(positionOut, want) {
			t.Errorf("ExamplePlayerPosition missing fragment %q", want)
		}
	}
}

func TestGenerateAccessorBodies(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Enums: []*ast.Enum{{
			Name: "PlayerStatus",
			Values: []*ast.EnumValue{
				{Name: "OFFLINE", Number: 0},
				{Name: "ONLINE", Number: 1},
			},
		}},
		Messages: []*ast.Message{{
			Name: "Player",
			Fields: []*ast.Field{
				{FieldType: "string", Name: "username", Number: 1},
				{FieldType: "PlayerStatus", Name: "status", Number: 2},
				{FieldType: "Position", Name: "position", Number: 3},
				{FieldType: "string", Name: "inventory", Number: 4, Repeated: true},
			},
			Maps: []*ast.MapField{{
				Name:      "stats",
				KeyType:   "string",
				ValueType: "int32",
				Number:    5,
			}},
			Oneofs: []*ast.Oneof{{
				Name: "contact",
				Fields: []*ast.Field{
					{FieldType: "string", Name: "email", Number: 6},
				},
			}},
			NestedMessages: []*ast.Message{{
				Name: "Position",
				Fields: []*ast.Field{
					{FieldType: "float", Name: "x", Number: 1},
				},
			}},
		}, {
			Name: "GameState",
			Fields: []*ast.Field{
				{FieldType: "Player", Name: "players", Number: 1, Repeated: true},
			},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	player := findFile(files, "ExamplePlayer")
	if player == nil {
		t.Fatalf("missing ExamplePlayer; got %v", classNames(files))
	}
	out := player.Source()
	for _, want := range []string{
		"func set_username(value: String) -> void:\n\t_username = value",
		"func get_username() -> String:\n\treturn _username",
		"var _status: ExamplePlayerStatus.PlayerStatus = 0 as ExamplePlayerStatus.PlayerStatus",
		"func set_status(value: ExamplePlayerStatus.PlayerStatus) -> void:\n\t_status = value",
		"func get_status() -> ExamplePlayerStatus.PlayerStatus:\n\treturn _status",
		"func new_position() -> ExamplePlayerPosition:\n\t_position = ExamplePlayerPosition.new()\n\treturn _position",
		"func add_inventory(value: String) -> void:\n\t_inventory.append(value)",
		"func add_stats(key: String, value: int) -> void:\n\t_stats[key] = value",
		"func get_stats() -> Dictionary[String, int]:\n\treturn _stats",
		"func set_email(value: String) -> void:\n\tif _oneof_contact != ContactOneOf.EMAIL:\n\t\t_oneof_contact = ContactOneOf.EMAIL\n\t_email = value",
		"func has_email() -> bool:\n\treturn _oneof_contact == ContactOneOf.EMAIL",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("ExamplePlayer missing fragment:\n%s\n--- output ---\n%s", want, out)
		}
	}

	gameState := findFile(files, "ExampleGameState")
	if gameState == nil {
		t.Fatalf("missing ExampleGameState; got %v", classNames(files))
	}
	gsOut := gameState.Source()
	if !strings.Contains(gsOut, "func add_players() -> ExamplePlayer:\n\tvar item: ExamplePlayer = ExamplePlayer.new()\n\t_players.append(item)\n\treturn item") {
		t.Errorf("GameState missing prefixed add_players accessor:\n%s", gsOut)
	}
}

func TestGenerateToBytesScalar(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Position",
			Fields: []*ast.Field{
				{FieldType: "float", Name: "x", Number: 1},
				{FieldType: "float", Name: "y", Number: 2},
				{FieldType: "float", Name: "z", Number: 3},
			},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "ExamplePosition")
	if f == nil {
		t.Fatalf("missing ExamplePosition; got %v", classNames(files))
	}
	out := f.Source()
	for _, want := range []string{
		"func to_bytes() -> PackedByteArray:",
		`"""Serialize message to bytes."""`,
		"var result: PackedByteArray = PackedByteArray()",
		"# Field x",
		"if _x != 0.0:",
		"result.append_array(ProtoCoreUtils.encode_varint(13))",
		"result.append_array(ProtoCoreUtils.encode_float(_x))",
		"# Field z",
		"if _z != 0.0:",
		"result.append_array(ProtoCoreUtils.encode_varint(29))",
		"return result",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("to_bytes scalar output missing fragment %q\n--- full output ---\n%s", want, out)
		}
	}
}

func TestGenerateToBytesStringRepeatedMessageOneofMap(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Player",
			Fields: []*ast.Field{
				{FieldType: "string", Name: "username", Number: 1},
				{FieldType: "Position", Name: "position", Number: 2},
				{FieldType: "string", Name: "inventory", Number: 3, Repeated: true},
			},
			Oneofs: []*ast.Oneof{{
				Name: "contact",
				Fields: []*ast.Field{
					{FieldType: "string", Name: "email", Number: 4},
				},
			}},
			Maps: []*ast.MapField{{
				Name:      "stats",
				KeyType:   "string",
				ValueType: "int32",
				Number:    5,
			}},
			NestedMessages: []*ast.Message{{
				Name: "Position",
				Fields: []*ast.Field{
					{FieldType: "float", Name: "x", Number: 1},
				},
			}},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "ExamplePlayer")
	if f == nil {
		t.Fatalf("missing ExamplePlayer; got %v", classNames(files))
	}
	out := f.Source()
	for _, want := range []string{
		"# Field username",
		`if _username != "":`,
		"var str_data: PackedByteArray = ProtoCoreUtils.encode_string(_username)",
		"# Field position",
		"if _position != null:",
		"var msg_data: PackedByteArray = _position.to_bytes()",
		"# Field inventory (repeated)",
		"for item in _inventory:",
		"# Field email",
		`if _email != "":`,
		"# Map field stats",
		"for key in _stats:",
		"var value: int = _stats[key]",
		"# Build map entry",
		"var entry: PackedByteArray = PackedByteArray()",
		"# Entry field 1: key",
		"var key_data: PackedByteArray = ProtoCoreUtils.encode_string(key)",
		"# Entry field 2: value",
		"entry.append_array(ProtoCoreUtils.encode_varint(value))",
		"# Append entry to result",
		"result.append_array(ProtoCoreUtils.encode_varint(42))",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("to_bytes complex output missing fragment %q\n--- full output ---\n%s", want, out)
		}
	}
}

func TestGenerateFromBytesScalar(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Position",
			Fields: []*ast.Field{
				{FieldType: "float", Name: "x", Number: 1},
				{FieldType: "float", Name: "y", Number: 2},
				{FieldType: "float", Name: "z", Number: 3},
			},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "ExamplePosition")
	if f == nil {
		t.Fatalf("missing ExamplePosition; got %v", classNames(files))
	}
	out := f.Source()
	for _, want := range []string{
		"func from_bytes(data: PackedByteArray) -> ProtoCoreUtils.ProtobufError:",
		`"""Deserialize message from bytes."""`,
		"var offset: int = 0",
		"while offset < data.size():",
		"# Read field tag",
		"var tag_result: Dictionary[String, int] = ProtoCoreUtils.decode_varint(data, offset)",
		`if tag_result["size"] == -1:`,
		"return ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND",
		"var field_number: int = ProtoCoreUtils.get_field_number(tag)",
		"var wire_type: int = ProtoCoreUtils.get_wire_type(tag)",
		"match field_number:",
		"# Field x",
		"if offset + 4 > data.size():",
		"return ProtoCoreUtils.ProtobufError.PARSE_INCOMPLETE",
		"_x = ProtoCoreUtils.decode_float(data, offset)",
		"offset += 4",
		"# Skip unknown field",
		"match wire_type:",
		"return ProtoCoreUtils.ProtobufError.UNDEFINED_STATE",
		"return ProtoCoreUtils.ProtobufError.NO_ERRORS",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("from_bytes scalar output missing fragment %q\n--- full output ---\n%s", want, out)
		}
	}
}

func TestGenerateFromBytesComplex(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Player",
			Fields: []*ast.Field{
				{FieldType: "string", Name: "username", Number: 1},
				{FieldType: "int32", Name: "level", Number: 2},
				{FieldType: "Position", Name: "position", Number: 3},
				{FieldType: "string", Name: "inventory", Number: 4, Repeated: true},
				{FieldType: "Player", Name: "friends", Number: 5, Repeated: true},
			},
			Maps: []*ast.MapField{{
				Name:      "stats",
				KeyType:   "string",
				ValueType: "int32",
				Number:    6,
			}},
		}, {
			// Sibling target for the Position reference above.
			Name: "Position",
			Fields: []*ast.Field{
				{FieldType: "float", Name: "x", Number: 1},
			},
		}},
	}
	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	f := findFile(files, "ExamplePlayer")
	if f == nil {
		t.Fatalf("missing ExamplePlayer; got %v", classNames(files))
	}
	out := f.Source()
	for _, want := range []string{
		"# Field username",
		"_username = ProtoCoreUtils.decode_string(data, offset, length)",
		"# Field level",
		"var result: Dictionary[String, int] = ProtoCoreUtils.decode_varint(data, offset)",
		`_level = result["value"]`,
		"# Field position",
		"_position = ExamplePosition.new()",
		"var msg_result: ProtoCoreUtils.ProtobufError = _position.from_bytes(msg_data)",
		"if msg_result != ProtoCoreUtils.ProtobufError.NO_ERRORS:",
		"# Field inventory",
		"_inventory.append(ProtoCoreUtils.decode_string(data, offset, length))",
		"# Field friends",
		"var msg_item: ExamplePlayer = ExamplePlayer.new()",
		"_friends.append(msg_item)",
		"# Map field stats",
		"var entry_data: PackedByteArray = data.slice(offset, offset + length)",
		"var entry_offset: int = 0",
		`var map_key: String = ""`,
		"var map_value: int = 0",
		"match entry_field_number:",
		"# Entry key",
		"map_key = ProtoCoreUtils.decode_string(entry_data, entry_offset, str_len)",
		"# Entry value",
		`map_value = result["value"]`,
		"_stats[map_key] = map_value",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("from_bytes complex output missing fragment %q\n--- full output ---\n%s", want, out)
		}
	}
}

func TestGenerateExampleStrictGDScriptShape(t *testing.T) {
	src, err := os.ReadFile("../../examples/example.proto")
	require.NoError(t, err)
	tokens, err := lexer.Tokenize(string(src), "example.proto")
	require.NoError(t, err)
	file, err := parser.Parse(tokens, "example.proto")
	require.NoError(t, err)
	require.Empty(t, validator.Validate(file, "example.proto"))
	files, err := generator.Generate(file, "example.proto", nil)
	require.NoError(t, err)

	untypedVar := regexp.MustCompile(`(?m)^\s*var\s+[A-Za-z_][A-Za-z0-9_]*\s*=`)
	bareDictionaryVar := regexp.MustCompile(`(?m)^\s*var\s+[A-Za-z_][A-Za-z0-9_]*:\s*Dictionary(\s|$)`)
	for _, f := range files {
		out := f.Source()
		assert.NotContains(t, out, ":=", "%s contains inferred declaration operator :=\n%s", f.Filename, firstMatchingLine(out, ":="))
		assert.Empty(t, untypedVar.FindString(out), "%s contains untyped local declaration", f.Filename)
		assert.Empty(t, bareDictionaryVar.FindString(out), "%s contains bare Dictionary local declaration", f.Filename)
		for _, forbidden := range []string{
			"result.size",
			"result.value",
			"tag_result.size",
			"tag_result.value",
			"length_result.size",
			"length_result.value",
			"entry_tag_result.size",
			"entry_tag_result.value",
			"len_result.size",
			"len_result.value",
		} {
			assert.NotContains(t, out, forbidden, "%s contains unsafe decode dictionary property access", f.Filename)
		}
		for _, forbidden := range []string{
			`int(num_result["value"])`,
			`int(enum_result["value"])`,
		} {
			assert.NotContains(t, out, forbidden, "%s passes Variant dictionary values directly to int()", f.Filename)
		}
	}
}

func TestProtoCoreUtilsStrictShape(t *testing.T) {
	out := generator.GenerateProtoCoreUtilsRaw()
	assert.NotContains(t, out, ":=", "proto_core_utils contains inferred declaration operator :=\n%s", firstMatchingLine(out, ":="))
	untypedVar := regexp.MustCompile(`(?m)^\s*var\s+[A-Za-z_][A-Za-z0-9_]*\s*=`)
	assert.Empty(t, untypedVar.FindString(out), "proto_core_utils contains untyped local declaration")
	bareDictionaryReturn := regexp.MustCompile(`(?m)->\s*Dictionary:`)
	assert.Empty(t, bareDictionaryReturn.FindString(out), "proto_core_utils contains bare Dictionary return type")
	discardedPackedByteArrayMutation := regexp.MustCompile(`(?m)^\s*result\.(append|resize)\(`)
	assert.Empty(t, discardedPackedByteArrayMutation.FindString(out), "proto_core_utils discards PackedByteArray mutation return value")
}

func firstMatchingLine(src, needle string) string {
	for _, line := range strings.Split(src, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func TestEnumFieldWireTypeIsVarint(t *testing.T) {
	src := `syntax = "proto3";
enum Platform {
	PLATFORM_UNSPECIFIED = 0;
	PLATFORM_DESKTOP = 1;
}
message Hello {
	string token = 1;
	Platform platform = 2;
}`
	tokens, err := lexer.Tokenize(src, "hello.proto")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(tokens, "hello.proto")
	if err != nil {
		t.Fatal(err)
	}
	if errs := validator.Validate(file, "hello.proto"); len(errs) != 0 {
		t.Fatalf("validation: %+v", errs)
	}
	files, err := generator.Generate(file, "hello.proto", nil)
	if err != nil {
		t.Fatal(err)
	}
	f := findFile(files, "HelloHello")
	if f == nil {
		t.Fatalf("missing HelloHello; got %v", classNames(files))
	}
	out := f.Source()
	// Field 2 with wire type 0 (varint) → tag = (2 << 3) | 0 = 16.
	if !strings.Contains(out, "encode_varint(16)") {
		t.Errorf("expected encode_varint(16) for enum field 2; output:\n%s", out)
	}
	if strings.Contains(out, "encode_varint(18)") {
		t.Errorf("enum field is using length-delimited wire type 2 (tag 18); should be varint (tag 16)")
	}
}

func TestGenerateImportedMessageUsesPrefixedClassName(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Uses",
			Fields: []*ast.Field{{
				FieldType:    "Shared",
				FullTypePath: "Shared",
				SourceFile:   "common.proto",
				Name:         "shared",
				Number:       1,
			}},
		}},
	}
	imported := &ast.ProtoFile{
		Syntax:   "proto3",
		Messages: []*ast.Message{{Name: "Shared"}},
	}

	files, err := generator.Generate(file, "main.proto", []generator.FileEntry{
		{File: imported, Filename: "common.proto"},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := findFile(files, "MainUses")
	if f == nil {
		t.Fatalf("missing MainUses; got %v", classNames(files))
	}
	got := f.Source()
	if !strings.Contains(got, "var _shared: CommonShared = null") {
		t.Fatalf("missing imported prefixed class type:\n%s", got)
	}
	if !strings.Contains(got, "_shared = CommonShared.new()") {
		t.Fatalf("missing imported prefixed constructor:\n%s", got)
	}
	if strings.Contains(got, "_shared = Shared.new()") {
		t.Fatalf("from_text uses unqualified Shared.new():\n%s", got)
	}
}

func TestGenerateImportedEnumFieldEmitsHelpers(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Uses",
			Fields: []*ast.Field{{
				FieldType:    "Color",
				FullTypePath: "shared.Color",
				SourceFile:   "shared.proto",
				IsEnum:       true,
				Name:         "color",
				Number:       1,
				EnumValues: []*ast.EnumValue{
					{Name: "COLOR_UNSPECIFIED", Number: 0},
					{Name: "RED", Number: 1},
					{Name: "BLUE", Number: 2},
				},
			}},
		}},
	}
	imported := &ast.ProtoFile{
		Syntax:  "proto3",
		Package: "shared",
		Enums: []*ast.Enum{{
			Name: "Color",
			Values: []*ast.EnumValue{
				{Name: "COLOR_UNSPECIFIED", Number: 0},
				{Name: "RED", Number: 1},
				{Name: "BLUE", Number: 2},
			},
		}},
	}

	files, err := generator.Generate(file, "main.proto", []generator.FileEntry{
		{File: imported, Filename: "shared.proto"},
	})
	if err != nil {
		t.Fatal(err)
	}
	f := findFile(files, "MainUses")
	if f == nil {
		t.Fatalf("missing MainUses; got %v", classNames(files))
	}
	got := f.Source()
	if !strings.Contains(got, "_get_enum_name_color") {
		t.Fatalf("missing _get_enum_name_color helper:\n%s", got)
	}
	if !strings.Contains(got, "_parse_enum_value_color") {
		t.Fatalf("missing _parse_enum_value_color helper:\n%s", got)
	}
	if !strings.Contains(got, `"RED"`) || !strings.Contains(got, `"BLUE"`) {
		t.Fatalf("imported enum values missing from helpers:\n%s", got)
	}
	// Imported enum is referenced as the wrapper class for that imported
	// file, qualified with the inner enum name: prefix from "shared.proto"
	// -> "Shared", concatenated with the type path "shared.Color" -> "Color"
	// => "SharedColor"; the enum inside that wrapper is "Color", giving
	// "SharedColor.Color.<VALUE>".
	if !strings.Contains(got, "SharedColor.Color.RED") {
		t.Fatalf("imported enum match patterns are not qualified by their wrapper class and inner enum:\n%s", got)
	}
	if strings.Contains(got, "\tColor.RED") || strings.Contains(got, " Color.RED") {
		t.Fatalf("unqualified Color.RED reference would not parse in GDScript:\n%s", got)
	}
}

func TestGenerateMapEnumUsesVarintPaths(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Enums: []*ast.Enum{{
			Name: "Color",
			Values: []*ast.EnumValue{
				{Name: "COLOR_UNSPECIFIED", Number: 0},
				{Name: "RED", Number: 1},
			},
		}},
		Messages: []*ast.Message{{
			Name: "Uses",
			Maps: []*ast.MapField{{
				Name:        "colors",
				KeyType:     "string",
				ValueType:   "Color",
				ValueIsEnum: true,
				Number:      1,
			}},
		}},
	}

	files, err := generator.Generate(file, "map_enum.proto", nil)
	if err != nil {
		t.Fatal(err)
	}
	f := findFile(files, "MapEnumUses")
	if f == nil {
		t.Fatalf("missing MapEnumUses; got %v", classNames(files))
	}
	got := f.Source()
	if !strings.Contains(got, "entry.append_array(ProtoCoreUtils.encode_varint(value))") {
		t.Fatalf("missing enum varint serialization path:\n%s", got)
	}
	if strings.Contains(got, "var value_msg_data: PackedByteArray = value.to_bytes()") {
		t.Fatalf("map enum value incorrectly serialized as message:\n%s", got)
	}
	if !strings.Contains(got, `map_value = result["value"] as MapEnumColor.Color`) {
		t.Fatalf("missing enum varint deserialization path:\n%s", got)
	}
}

func TestGenerateMessageEnumNameCollisionDoesNotUseEnumPaths(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{
			{
				Name: "A",
				NestedEnums: []*ast.Enum{{
					Name: "Status",
					Values: []*ast.EnumValue{
						{Name: "STATUS_UNSPECIFIED", Number: 0},
					},
				}},
			},
			{
				Name:           "B",
				NestedMessages: []*ast.Message{{Name: "Status"}},
				Fields: []*ast.Field{{
					FieldType:    "Status",
					FullTypePath: "B.Status",
					Name:         "status",
					Number:       1,
				}},
			},
		},
	}

	files, err := generator.Generate(file, "collision.proto", nil)
	if err != nil {
		t.Fatal(err)
	}
	f := findFile(files, "CollisionB")
	if f == nil {
		t.Fatalf("missing CollisionB; got %v", classNames(files))
	}
	got := f.Source()
	if !strings.Contains(got, "var _status: CollisionBStatus = null") {
		t.Fatalf("message field should default to null with prefixed type:\n%s", got)
	}
	if !strings.Contains(got, "func new_status() -> CollisionBStatus:") {
		t.Fatalf("message field should use new_ accessor with prefixed type:\n%s", got)
	}
	if strings.Contains(got, "func set_status(value: CollisionBStatus) -> void:") {
		t.Fatalf("message field incorrectly treated as enum/scalar:\n%s", got)
	}
	if strings.Contains(got, "encode_varint(_status)") {
		t.Fatalf("message field incorrectly serialized as enum varint:\n%s", got)
	}
}

func TestGenerateExampleGoldenDirectory(t *testing.T) {
	src, err := os.ReadFile("../../examples/example.proto")
	if err != nil {
		t.Fatalf("read example.proto: %v", err)
	}
	tokens, err := lexer.Tokenize(string(src), "example.proto")
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	file, err := parser.Parse(tokens, "example.proto")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if errs := validator.Validate(file, "example.proto"); len(errs) != 0 {
		t.Fatalf("validation errors: %+v", errs)
	}

	files, err := generator.Generate(file, "example.proto", nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	goldenDir := "../../examples/golden"
	seen := map[string]bool{}
	for _, f := range files {
		seen[f.Filename] = true
		path := filepath.Join(goldenDir, f.Filename)
		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("missing golden for %s: %v\n--- got ---\n%s", f.Filename, err, f.Source())
			continue
		}
		got := f.Source()
		if got == string(want) {
			continue
		}
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(string(want), "\n")
		for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
			if gotLines[i] != wantLines[i] {
				t.Errorf("%s first diff at line %d:\n  got:  %q\n  want: %q",
					f.Filename, i+1, gotLines[i], wantLines[i])
				break
			}
		}
		if len(gotLines) != len(wantLines) {
			t.Errorf("%s line count mismatch: got %d, want %d", f.Filename, len(gotLines), len(wantLines))
		}
	}
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("read golden dir: %v", err)
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		if !seen[ent.Name()] {
			t.Errorf("golden file not produced: %s", ent.Name())
		}
	}
}

func TestProtoCoreUtilsGolden(t *testing.T) {
	got := generator.GenerateProtoCoreUtilsRaw()
	wantBytes, err := os.ReadFile("../../examples/proto_core_utils_golden.gd")
	if err != nil {
		t.Fatalf("read proto_core_utils_golden.gd: %v", err)
	}
	want := string(wantBytes)
	if got == want {
		return
	}
	gotLines := strings.Split(got, "\n")
	wantLines := strings.Split(want, "\n")
	for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
		if gotLines[i] != wantLines[i] {
			t.Errorf("first diff at line %d:\n  got:  %q\n  want: %q", i+1, gotLines[i], wantLines[i])
			break
		}
	}
	if len(gotLines) != len(wantLines) {
		t.Errorf("line counts: got %d, want %d", len(gotLines), len(wantLines))
	}
}

func TestGenerateDetectsWithinFileClassNameCollision(t *testing.T) {
	src := `syntax = "proto3";
message FooBar { int32 a = 1; }
message Foo { message Bar { int32 b = 1; } }
`
	tokens, err := lexer.Tokenize(src, "collide.proto")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(tokens, "collide.proto")
	if err != nil {
		t.Fatal(err)
	}

	_, err = generator.Generate(file, "collide.proto", nil)
	if err == nil {
		t.Fatal("expected collision error")
	}
	msg := err.Error()
	for _, want := range []string{"class name collision", "FooBar", "Foo.Bar", "CollideFooBar"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q: %s", want, msg)
		}
	}
}

func TestGenerateDetectsEnumMessageClassNameCollision(t *testing.T) {
	src := `syntax = "proto3";
enum Status { OK = 0; }
message Status { int32 a = 1; }
`
	tokens, err := lexer.Tokenize(src, "collide_enum.proto")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(tokens, "collide_enum.proto")
	if err != nil {
		t.Fatal(err)
	}

	_, err = generator.Generate(file, "collide_enum.proto", nil)
	if err == nil {
		t.Fatal("expected collision error")
	}
	if !strings.Contains(err.Error(), "class name collision") {
		t.Errorf("missing collision marker: %s", err.Error())
	}
}

// TestGenerateUnresolvedCrossFileTypeIsError covers the defensive path in
// renderedType: a field with SourceFile set to a file that was never threaded
// through Generate's imports must surface an explicit error rather than
// silently emitting a guessed class name.
func TestGenerateUnresolvedCrossFileTypeIsError(t *testing.T) {
	file := &ast.ProtoFile{
		Syntax: "proto3",
		Messages: []*ast.Message{{
			Name: "Uses",
			Fields: []*ast.Field{{
				FieldType:    "Stranger",
				FullTypePath: "Stranger",
				SourceFile:   "unknown.proto",
				Name:         "x",
				Number:       1,
			}},
		}},
	}

	_, err := generator.Generate(file, "main.proto", nil)
	if err == nil {
		t.Fatal("expected error from unresolved cross-file reference")
	}
	if !strings.Contains(err.Error(), "no resolver entry") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "unknown.proto") {
		t.Fatalf("error missing source filename: %v", err)
	}
}
