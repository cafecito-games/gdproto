package generator_test

import (
	"strings"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/generator"
)

func TestGenerateEmptyProto(t *testing.T) {
	file := &ast.ProtoFile{Syntax: "proto3"}
	cls, err := generator.Generate(file, "example.proto")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if cls == nil {
		t.Fatal("Generate returned nil ClassDefinition")
	}
	if cls.ClassNameDirective != "Example" {
		t.Errorf("ClassNameDirective = %q, want %q", cls.ClassNameDirective, "Example")
	}
	if cls.Extends != "RefCounted" {
		t.Errorf("Extends = %q, want %q", cls.Extends, "RefCounted")
	}
	out := cls.ToGDScript(0)
	wantPrefix := "class_name Example\n\nextends RefCounted\n\nenum ProtobufError {"
	if !strings.HasPrefix(out, wantPrefix) {
		t.Errorf("output does not start with expected prefix; got first 200 chars:\n%s", out[:minInt(len(out), 200)])
	}
	if !strings.Contains(out, "static func encode_varint(") {
		t.Errorf("output missing protobuf_core helpers")
	}
	if !strings.Contains(out, "static func escape_string_text_format(") {
		t.Errorf("output missing text format helpers")
	}
}

func TestStemToClassName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "example.proto", "Example"},
		{"snake_case", "snake_case_name.proto", "SnakeCaseName"},
		{"with dir", "/tmp/foo/bar_baz.proto", "BarBaz"},
		{"single", "x.proto", "X"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := &ast.ProtoFile{Syntax: "proto3"}
			cls, err := generator.Generate(file, tc.in)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if cls.ClassNameDirective != tc.want {
				t.Errorf("ClassNameDirective = %q, want %q", cls.ClassNameDirective, tc.want)
			}
		})
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
