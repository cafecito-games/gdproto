package gdast

import "strings"

// Node is implemented by every GDScript AST node. ToGDScript renders the node
// at the given indentation level (each level is one tab character).
type Node interface {
	ToGDScript(indentLevel int) string
}

// Expression is a marker interface for nodes that can appear in expression
// position.
type Expression interface {
	Node
	expression()
}

// Statement is a marker interface for nodes that can appear in statement
// position.
type Statement interface {
	Node
	statement()
}

func indent(level int) string {
	return strings.Repeat("\t", level)
}
