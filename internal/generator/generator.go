package generator

import (
	"path/filepath"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// Generate produces a gdast ClassDefinition representing the GDScript
// translation of the proto file. The sourceName is the input filename used in
// the file header comment.
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
		constantsSectionStatement(),
		errorEnumStatement(),
	}

	if len(g.file.Enums) > 0 {
		statements = append(statements, sectionHeaderStatement("Enums"))
		for _, e := range g.file.Enums {
			statements = append(statements, generateEnum(e))
		}
	}

	if len(g.file.Messages) > 0 {
		statements = append(statements, sectionHeaderStatement("Messages"))
		for _, m := range g.file.Messages {
			statements = append(statements, g.generateMessage(m))
		}
	}

	return &gdast.ClassDefinition{
		HeaderComment: headerCommentText(headerSourceName(g.sourceName)),
		Extends:       "RefCounted",
		Statements:    statements,
	}, nil
}

// headerSourceName returns the basename of the input path used in the file
// header comment (e.g. "/tmp/foo/example.proto" -> "example.proto").
func headerSourceName(filename string) string {
	return filepath.Base(filename)
}
