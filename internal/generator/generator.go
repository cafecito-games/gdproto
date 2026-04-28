package generator

import (
	"path/filepath"
	"strings"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// Generate produces a gdast ClassDefinition representing the GDScript
// translation of the proto file. The sourceName is the input filename and is
// used to derive the class_name directive (basename without extension,
// converted from snake_case to PascalCase).
func Generate(file *ast.ProtoFile, sourceName string) (*gdast.ClassDefinition, error) {
	g := &generator{file: file, sourceName: sourceName, enumTypes: map[string]bool{}}
	g.collectEnumTypes()
	return g.generate()
}

type generator struct {
	file       *ast.ProtoFile
	sourceName string
	enumTypes  map[string]bool
}

func (g *generator) collectEnumTypes() {
	for _, e := range g.file.Enums {
		g.enumTypes[e.Name] = true
	}
	for _, m := range g.file.Messages {
		g.collectMessageEnumTypes(m)
	}
}

func (g *generator) collectMessageEnumTypes(m *ast.Message) {
	for _, e := range m.NestedEnums {
		g.enumTypes[e.Name] = true
	}
	for _, nested := range m.NestedMessages {
		g.collectMessageEnumTypes(nested)
	}
}

func (g *generator) generate() (*gdast.ClassDefinition, error) {
	statements := []gdast.Node{
		errorEnumStatement(),
		gdast.EmptyLine{},
		protobufCoreStatement(),
		gdast.EmptyLine{},
		textFormatUtilsStatement(),
	}

	for _, m := range g.file.Messages {
		statements = append(statements, gdast.EmptyLine{}, generateMessageStub(m))
	}

	return &gdast.ClassDefinition{
		ClassNameDirective: stemToClassName(g.sourceName),
		Extends:            "RefCounted",
		Statements:         statements,
	}, nil
}

// stemToClassName converts a proto file path to a PascalCase class name based
// on its basename (e.g. "example.proto" -> "Example",
// "snake_case_name.proto" -> "SnakeCaseName").
func stemToClassName(filename string) string {
	base := filepath.Base(filename)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	parts := strings.Split(stem, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}
