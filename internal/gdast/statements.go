package gdast

import "strings"

// EmptyLine renders as a literal empty line regardless of indentation.
type EmptyLine struct{}

// ToGDScript always returns the empty string.
func (EmptyLine) ToGDScript(int) string { return "" }
func (EmptyLine) statement()            {}

// PassStatement renders as `pass` at the requested indentation.
type PassStatement struct{}

// ToGDScript renders the `pass` keyword at the given indent level.
func (PassStatement) ToGDScript(level int) string { return indent(level) + "pass" }
func (PassStatement) statement()                  {}

// BreakStatement renders as `break` at the requested indentation.
type BreakStatement struct{}

// ToGDScript renders the `break` keyword at the given indent level.
func (BreakStatement) ToGDScript(level int) string { return indent(level) + "break" }
func (BreakStatement) statement()                  {}

// ContinueStatement renders as `continue` at the requested indentation.
type ContinueStatement struct{}

// ToGDScript renders the `continue` keyword at the given indent level.
func (ContinueStatement) ToGDScript(level int) string { return indent(level) + "continue" }
func (ContinueStatement) statement()                  {}

// RawStatement embeds an opaque snippet of GDScript source. Each non-blank
// line is prefixed with the requested indentation; blank lines remain empty.
type RawStatement struct {
	Code string
}

// ToGDScript prefixes every non-blank line of Code with the requested
// indentation, preserving relative tabs already present in the snippet.
func (r RawStatement) ToGDScript(level int) string {
	if r.Code == "" {
		return ""
	}
	if level == 0 {
		return r.Code
	}
	lines := strings.Split(r.Code, "\n")
	pad := indent(level)
	out := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			out[i] = line
		} else {
			out[i] = pad + line
		}
	}
	return strings.Join(out, "\n")
}

func (RawStatement) statement() {}

// DocString renders a triple-quoted GDScript doc string at the requested
// indentation. The text content is emitted verbatim between the delimiters.
type DocString struct {
	Text string
}

// ToGDScript wraps the text in triple double quotes prefixed by the indent.
func (d DocString) ToGDScript(level int) string {
	return indent(level) + `"""` + d.Text + `"""`
}

func (DocString) statement() {}

// Comment renders a `#` comment block. Single-line text becomes a single
// `# text` line; multi-line text emits one `# line` per line, with empty
// lines collapsed to a bare `#`. Leading and trailing whitespace are stripped
// from the overall block.
type Comment struct {
	Text string
}

// ToGDScript renders the comment using `# ` prefixes at the given indent.
func (c Comment) ToGDScript(level int) string {
	return renderCommentBlock(c.Text, level, "#")
}

func (Comment) statement() {}

// DocumentationComment renders a `##` documentation comment block following
// the same rules as Comment, but using the `##` prefix.
type DocumentationComment struct {
	Text string
}

// ToGDScript renders the documentation comment using `## ` prefixes.
func (d DocumentationComment) ToGDScript(level int) string {
	return renderCommentBlock(d.Text, level, "##")
}

func (DocumentationComment) statement() {}

func renderCommentBlock(text string, level int, marker string) string {
	text = strings.TrimSpace(text)
	pad := indent(level)
	if !strings.Contains(text, "\n") {
		return pad + marker + " " + text
	}
	lines := strings.Split(text, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			out[i] = pad + marker
		} else {
			out[i] = pad + marker + " " + line
		}
	}
	return strings.Join(out, "\n")
}

// ExpressionStatement wraps a bare expression so it can be used where a
// Statement is required (for example, top-level call statements).
type ExpressionStatement struct {
	Expression Expression
}

// ToGDScript emits the wrapped expression at the requested indentation.
func (e ExpressionStatement) ToGDScript(level int) string {
	return indent(level) + e.Expression.ToGDScript(0)
}

func (ExpressionStatement) statement() {}

// VarDeclaration renders a `var` (or `const` when IsConst is true) declaration.
// When TypeHint is set the assignment uses `=`; when only InitialValue is set
// the declaration uses `:=` to request type inference.
type VarDeclaration struct {
	Name         string
	TypeHint     string
	InitialValue Expression
	IsConst      bool
}

// ToGDScript emits the declaration with appropriate type hint and operator.
func (v VarDeclaration) ToGDScript(level int) string {
	keyword := "var"
	if v.IsConst {
		keyword = "const"
	}
	result := indent(level) + keyword + " " + v.Name
	if v.TypeHint != "" {
		result += ": " + v.TypeHint
	}
	if v.InitialValue != nil {
		if v.TypeHint != "" {
			result += " = " + v.InitialValue.ToGDScript(0)
		} else {
			result += " := " + v.InitialValue.ToGDScript(0)
		}
	}
	return result
}

func (VarDeclaration) statement() {}

// Assignment renders an assignment statement. Operator defaults to `=` when
// empty; compound operators such as `+=` may be supplied directly.
type Assignment struct {
	Target   Expression
	Value    Expression
	Operator string
}

// ToGDScript emits `target op value` at the requested indentation.
func (a Assignment) ToGDScript(level int) string {
	op := a.Operator
	if op == "" {
		op = "="
	}
	return indent(level) + a.Target.ToGDScript(0) + " " + op + " " + a.Value.ToGDScript(0)
}

func (Assignment) statement() {}

// ReturnStatement renders a `return` statement, optionally with a value.
type ReturnStatement struct {
	Value Expression
}

// ToGDScript emits `return` or `return value` at the requested indentation.
func (r ReturnStatement) ToGDScript(level int) string {
	if r.Value == nil {
		return indent(level) + "return"
	}
	return indent(level) + "return " + r.Value.ToGDScript(0)
}

func (ReturnStatement) statement() {}
