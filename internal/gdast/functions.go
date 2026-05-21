package gdast

import (
	"strconv"
	"strings"
)

// GDScript is the root container for a GDScript file. Items are joined with
// newlines when rendered.
type GDScript struct {
	Items []Node
}

// ToGDScript renders each item on its own line at the given indent level.
func (s GDScript) ToGDScript(indentLevel int) string {
	var sb strings.Builder
	for i, item := range s.Items {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(item.ToGDScript(indentLevel))
	}
	return sb.String()
}

// Parameter represents a function or signal parameter. It is rendered inline
// by the containing Function or SignalDefinition rather than as a Node.
type Parameter struct {
	Name     string
	TypeHint string
	Default  Expression
}

// Render returns the textual form of the parameter (no indentation): name,
// optionally followed by a type hint and/or default value.
func (p Parameter) Render() string {
	switch {
	case p.TypeHint != "" && p.Default != nil:
		return p.Name + ": " + p.TypeHint + " = " + p.Default.ToGDScript(0)
	case p.TypeHint != "":
		return p.Name + ": " + p.TypeHint
	case p.Default != nil:
		return p.Name + " = " + p.Default.ToGDScript(0)
	default:
		return p.Name
	}
}

func renderParameters(params []Parameter) string {
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = p.Render()
	}
	return strings.Join(parts, ", ")
}

// Function defines a GDScript function with optional parameters, return type,
// and static modifier. An empty Body emits a `pass` statement.
type Function struct {
	Name       string
	Parameters []Parameter
	Body       []Statement
	ReturnType string
	IsStatic   bool
}

// ToGDScript renders the function signature and indented body.
func (f Function) ToGDScript(level int) string {
	pad := indent(level)
	var header strings.Builder
	header.WriteString(pad)
	if f.IsStatic {
		header.WriteString("static ")
	}
	header.WriteString("func ")
	header.WriteString(f.Name)
	header.WriteByte('(')
	header.WriteString(renderParameters(f.Parameters))
	header.WriteByte(')')
	if f.ReturnType != "" {
		header.WriteString(" -> ")
		header.WriteString(f.ReturnType)
	}
	header.WriteByte(':')

	lines := []string{header.String()}
	if len(f.Body) == 0 {
		lines = append(lines, indent(level+1)+"pass")
	} else {
		for _, stmt := range f.Body {
			lines = append(lines, stmt.ToGDScript(level+1))
		}
	}
	return strings.Join(lines, "\n")
}

func (Function) statement() {}

// EnumValue is a single entry within an EnumDefinition. A nil Value means the
// member uses GDScript auto-incrementation.
type EnumValue struct {
	Name  string
	Value *int
}

// EnumDefinition renders a GDScript `enum` block. An empty Name produces an
// anonymous enum.
type EnumDefinition struct {
	Name   string
	Values []EnumValue
}

// ToGDScript renders the enum across multiple lines with comma separators.
func (e EnumDefinition) ToGDScript(level int) string {
	pad := indent(level)
	header := pad + "enum"
	if e.Name != "" {
		header += " " + e.Name
	}
	header += " {"

	lines := []string{header}
	innerPad := indent(level + 1)
	for i, v := range e.Values {
		line := innerPad + v.Name
		if v.Value != nil {
			line += " = " + strconv.Itoa(*v.Value)
		}
		if i < len(e.Values)-1 {
			line += ","
		}
		lines = append(lines, line)
	}
	lines = append(lines, pad+"}")
	return strings.Join(lines, "\n")
}

func (EnumDefinition) statement() {}

// SignalDefinition renders a GDScript `signal` declaration with optional
// parameters.
type SignalDefinition struct {
	Name       string
	Parameters []Parameter
}

// ToGDScript renders `signal name` or `signal name(params)`.
func (s SignalDefinition) ToGDScript(level int) string {
	pad := indent(level)
	if len(s.Parameters) == 0 {
		return pad + "signal " + s.Name
	}
	return pad + "signal " + s.Name + "(" + renderParameters(s.Parameters) + ")"
}

func (SignalDefinition) statement() {}

// ClassDefinition models either a top-level GDScript file (when Name is empty)
// or a nested `class` block. Top-level definitions emit `class_name` and
// `extends` directives at the file scope, followed by an optional header
// comment block. Statements within a top-level class are separated by three
// blank lines to match the gdproto wrapper layout; nested classes use a single
// blank line between statements.
type ClassDefinition struct {
	Name               string
	Extends            string
	ClassNameDirective string
	HeaderComment      string
	Statements         []Node
	// TightStatements disables the automatic blank line that would otherwise
	// be inserted between adjacent statements. When true, the caller controls
	// spacing entirely via EmptyLine entries in Statements.
	TightStatements bool
}

// suppressAutoBlank reports whether the automatic blank line between two
// adjacent statements should be omitted. Field declarations group tightly with
// each other and with their immediately preceding section comment.
func suppressAutoBlank(prev, next Node) bool {
	_, nextIsVar := next.(VarDeclaration)
	if !nextIsVar {
		return false
	}
	switch prev.(type) {
	case VarDeclaration, Comment:
		return true
	}
	return false
}

// ToGDScript renders the class header (if any) followed by the body.
func (c ClassDefinition) ToGDScript(level int) string {
	var lines []string
	pad := indent(level)
	bodyIndent := level

	if c.Name == "" {
		if c.ClassNameDirective != "" {
			lines = append(lines, "class_name "+c.ClassNameDirective, "")
		}
		if c.Extends != "" {
			lines = append(lines, "extends "+c.Extends, "")
		}
		if c.HeaderComment != "" {
			lines = append(lines, renderCommentBlock(c.HeaderComment, 0, "#"), "")
		}
	} else {
		header := pad + "class " + c.Name
		if c.Extends != "" {
			header += " extends " + c.Extends
		}
		header += ":"
		lines = append(lines, header)
		bodyIndent = level + 1
	}

	for i, stmt := range c.Statements {
		lines = append(lines, stmt.ToGDScript(bodyIndent))
		if !c.TightStatements && i < len(c.Statements)-1 && !suppressAutoBlank(stmt, c.Statements[i+1]) {
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}
