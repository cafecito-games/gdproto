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
