package generator

import (
	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// generateMessageStub returns a minimal class definition for a message. T1+
// replaces this with full generation including fields, accessors, and
// serialization methods.
func generateMessageStub(m *ast.Message) *gdast.ClassDefinition {
	return &gdast.ClassDefinition{
		Name:       m.Name,
		Extends:    "RefCounted",
		Statements: []gdast.Node{gdast.PassStatement{}},
	}
}
