package importer_test

import (
	"fmt"
	"testing"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/importer"
	"github.com/cafecito-games/gogdproto/internal/lexer"
	"github.com/cafecito-games/gogdproto/internal/parser"
)

// memFS is a test-only in-memory FS.
type memFS struct{ files map[string]string }

func (m *memFS) Read(path string) ([]byte, error) {
	if c, ok := m.files[path]; ok {
		return []byte(c), nil
	}
	return nil, fmt.Errorf("not found: %s", path)
}

func (m *memFS) Exists(path string) bool { _, ok := m.files[path]; return ok }

// parseFile is a helper.
func parseFile(t *testing.T, src string) *ast.ProtoFile {
	t.Helper()
	tokens, err := lexer.Tokenize(src, "in.proto")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(tokens, "in.proto")
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func findField(t *testing.T, file *ast.ProtoFile, message, field string) *ast.Field {
	t.Helper()
	for _, m := range file.Messages {
		if m.Name != message {
			continue
		}
		for _, f := range m.Fields {
			if f.Name == field {
				return f
			}
		}
	}
	t.Fatalf("field %s.%s not found", message, field)
	return nil
}

func findMap(t *testing.T, file *ast.ProtoFile, message, name string) *ast.MapField {
	t.Helper()
	for _, m := range file.Messages {
		if m.Name != message {
			continue
		}
		for _, mp := range m.Maps {
			if mp.Name == name {
				return mp
			}
		}
	}
	t.Fatalf("map %s.%s not found", message, name)
	return nil
}

func TestResolveExternal_ImportEnum(t *testing.T) {
	other := `syntax = "proto3"; enum E { A = 0; B = 1; }`
	in := `syntax = "proto3";
import "other.proto";
message M {
    E e = 1;
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	f := findField(t, file, "M", "e")
	if f.SourceFile != "other.proto" {
		t.Errorf("SourceFile = %q, want %q", f.SourceFile, "other.proto")
	}
	if !f.IsEnum {
		t.Errorf("IsEnum = false, want true")
	}
}

func TestResolveExternal_ImportMessage(t *testing.T) {
	other := `syntax = "proto3"; message Outer { int32 x = 1; }`
	in := `syntax = "proto3";
import "other.proto";
message M {
    Outer o = 1;
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	f := findField(t, file, "M", "o")
	if f.SourceFile != "other.proto" {
		t.Errorf("SourceFile = %q", f.SourceFile)
	}
	if f.IsEnum {
		t.Errorf("IsEnum = true, want false")
	}
}

func TestResolveExternal_PackageRelativeUnqualified(t *testing.T) {
	other := `syntax = "proto3"; package shared; enum E { A = 0; }`
	in := `syntax = "proto3";
package shared;
import "other.proto";
message M {
    E e1 = 1;
    shared.E e2 = 2;
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	f1 := findField(t, file, "M", "e1")
	if f1.SourceFile != "other.proto" || !f1.IsEnum {
		t.Errorf("e1: SourceFile=%q IsEnum=%v", f1.SourceFile, f1.IsEnum)
	}
	f2 := findField(t, file, "M", "e2")
	if f2.SourceFile != "other.proto" || !f2.IsEnum {
		t.Errorf("e2: SourceFile=%q IsEnum=%v", f2.SourceFile, f2.IsEnum)
	}
}

func TestResolveExternal_MapValueImported(t *testing.T) {
	other := `syntax = "proto3"; enum Color { RED = 0; BLUE = 1; }`
	in := `syntax = "proto3";
import "other.proto";
message M {
    map<string, Color> m = 1;
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	mp := findMap(t, file, "M", "m")
	if mp.ValueSourceFile != "other.proto" {
		t.Errorf("ValueSourceFile = %q", mp.ValueSourceFile)
	}
	if !mp.ValueIsEnum {
		t.Errorf("ValueIsEnum = false, want true")
	}
}

func TestResolveExternal_NestedTypeImport(t *testing.T) {
	other := `syntax = "proto3"; message Outer { enum Inner { A = 0; B = 1; } }`
	in := `syntax = "proto3";
import "other.proto";
message M {
    Outer.Inner i = 1;
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	f := findField(t, file, "M", "i")
	if f.SourceFile != "other.proto" {
		t.Errorf("SourceFile = %q", f.SourceFile)
	}
	if !f.IsEnum {
		t.Errorf("IsEnum = false, want true")
	}
}

func TestResolveExternal_MissingImportSilent(t *testing.T) {
	in := `syntax = "proto3";
import "missing.proto";
message M {
    int32 x = 1;
}`
	fs := &memFS{files: map[string]string{}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
