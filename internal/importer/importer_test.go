package importer_test

import (
	"fmt"
	"os"
	"path/filepath"
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

func TestResolveExternal_OneofImported(t *testing.T) {
	other := `syntax = "proto3"; enum Kind { K0 = 0; K1 = 1; } message Payload { int32 v = 1; }`
	in := `syntax = "proto3";
import "other.proto";
message M {
    oneof choice {
        Kind k = 1;
        Payload p = 2;
    }
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	var kField, pField *ast.Field
	for _, m := range file.Messages {
		for _, o := range m.Oneofs {
			for _, f := range o.Fields {
				switch f.Name {
				case "k":
					kField = f
				case "p":
					pField = f
				}
			}
		}
	}
	if kField == nil || kField.SourceFile != "other.proto" || !kField.IsEnum {
		t.Errorf("k: %+v", kField)
	}
	if pField == nil || pField.SourceFile != "other.proto" || pField.IsEnum {
		t.Errorf("p: %+v", pField)
	}
}

func TestResolveExternal_NestedMessageAnnotated(t *testing.T) {
	other := `syntax = "proto3"; enum E { A = 0; }`
	in := `syntax = "proto3";
import "other.proto";
message Outer {
    message Inner {
        E e = 1;
    }
}`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := parseFile(t, in)
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	inner := file.Messages[0].NestedMessages[0]
	f := inner.Fields[0]
	if f.SourceFile != "other.proto" || !f.IsEnum {
		t.Errorf("got %+v", f)
	}
}

func TestResolveExternal_FullTypePathFallback(t *testing.T) {
	// Field already has FullTypePath set but its raw FieldType won't
	// match the lookup; resolution should fall back to FullTypePath.
	other := `syntax = "proto3"; package shared; enum E { A = 0; }`
	fs := &memFS{files: map[string]string{"other.proto": other}}
	file := &ast.ProtoFile{
		Position: ast.Position{Line: 1, Column: 1},
		Syntax:   "proto3",
		Imports:  []*ast.Import{{Position: ast.Position{Line: 2, Column: 1}, Path: "other.proto"}},
		Messages: []*ast.Message{
			{
				Position: ast.Position{Line: 3, Column: 1},
				Name:     "M",
				Fields: []*ast.Field{
					{
						Position:     ast.Position{Line: 4, Column: 1},
						FieldType:    "UnknownAlias",
						FullTypePath: "shared.E",
						Name:         "f",
						Number:       1,
					},
				},
			},
		},
	}
	if err := importer.ResolveExternal(file, "in.proto", fs); err != nil {
		t.Fatal(err)
	}
	f := file.Messages[0].Fields[0]
	if f.SourceFile != "other.proto" || !f.IsEnum {
		t.Errorf("expected fallback resolution; got %+v", f)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestOSFSReadAndExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.proto")
	if err := os.WriteFile(path, []byte(`syntax = "proto3";`), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := &importer.OSFS{BaseDir: dir}
	if !fs.Exists("a.proto") {
		t.Errorf("Exists(a.proto) = false")
	}
	data, err := fs.Read("a.proto")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(data) != `syntax = "proto3";` {
		t.Errorf("Read returned %q", data)
	}
	if fs.Exists("missing.proto") {
		t.Errorf("Exists(missing.proto) = true")
	}
	if _, err := fs.Read("missing.proto"); err == nil {
		t.Errorf("Read(missing.proto) returned no error")
	}
}

func TestOSFSWalkUp(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "parent.proto"), []byte(`syntax = "proto3"; enum E { A = 0; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childPath := filepath.Join(subDir, "child.proto")
	if err := os.WriteFile(childPath, []byte(`syntax = "proto3"; import "parent.proto"; message M { E e = 1; }`), 0o644); err != nil {
		t.Fatal(err)
	}

	tokens, err := lexer.Tokenize(string(must(os.ReadFile(childPath))), childPath)
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.Parse(tokens, childPath)
	if err != nil {
		t.Fatal(err)
	}

	fs := &importer.OSFS{BaseDir: subDir}
	if err := importer.ResolveExternal(file, childPath, fs); err != nil {
		t.Fatal(err)
	}
	f := file.Messages[0].Fields[0]
	if f.SourceFile == "" || !f.IsEnum {
		t.Errorf("expected import resolution; got %+v", f)
	}
}

func TestOSFSWalkUpBasename(t *testing.T) {
	// Place the target file two levels above BaseDir under a different
	// relative path so only the basename strategy can match.
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	// Target at dir/shared.proto; import path "nested/shared.proto"
	// won't match dir+path, but basename "shared.proto" will.
	if err := os.WriteFile(filepath.Join(dir, "shared.proto"), []byte(`syntax = "proto3";`), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := &importer.OSFS{BaseDir: deep}
	if !fs.Exists("nested/shared.proto") {
		t.Errorf("expected basename walk-up to find shared.proto")
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
