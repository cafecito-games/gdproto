package gdast

import (
	"fmt"
	"strings"
)

// ElifBranch is a single `elif` clause within an IfStatement.
type ElifBranch struct {
	Condition Expression
	Body      []Statement
}

// IfStatement renders an `if` block, optionally followed by any number of
// `elif` branches and an `else` body. ElseBody is nil when no else clause
// is present.
type IfStatement struct {
	Condition    Expression
	Body         []Statement
	ElifBranches []ElifBranch
	ElseBody     []Statement
}

// ToGDScript renders the if/elif/else chain at the requested indentation.
func (i IfStatement) ToGDScript(level int) string {
	pad := indent(level)
	var lines []string
	lines = append(lines, pad+"if "+i.Condition.ToGDScript(0)+":")
	for _, s := range i.Body {
		lines = append(lines, s.ToGDScript(level+1))
	}
	for _, eb := range i.ElifBranches {
		lines = append(lines, pad+"elif "+eb.Condition.ToGDScript(0)+":")
		for _, s := range eb.Body {
			lines = append(lines, s.ToGDScript(level+1))
		}
	}
	if i.ElseBody != nil {
		lines = append(lines, pad+"else:")
		for _, s := range i.ElseBody {
			lines = append(lines, s.ToGDScript(level+1))
		}
	}
	return strings.Join(lines, "\n")
}

func (IfStatement) statement() {}

// WhileStatement renders a `while` loop. An empty Body emits a single
// indented `pass` to keep the block syntactically valid.
type WhileStatement struct {
	Condition Expression
	Body      []Statement
}

// ToGDScript renders the while loop at the requested indentation.
func (w WhileStatement) ToGDScript(level int) string {
	pad := indent(level)
	lines := []string{pad + "while " + w.Condition.ToGDScript(0) + ":"}
	if len(w.Body) == 0 {
		lines = append(lines, indent(level+1)+"pass")
	} else {
		for _, s := range w.Body {
			lines = append(lines, s.ToGDScript(level+1))
		}
	}
	return strings.Join(lines, "\n")
}

func (WhileStatement) statement() {}

// ForStatement renders a `for ... in ...` loop. When TypeHint is non-empty
// the loop variable is declared with `var name: Type`; otherwise the bare
// variable name is used. An empty Body emits a single indented `pass`.
type ForStatement struct {
	Variable string
	Iterable Expression
	Body     []Statement
	TypeHint string
}

// ToGDScript renders the for loop at the requested indentation.
func (f ForStatement) ToGDScript(level int) string {
	pad := indent(level)
	var varDeclaration string
	if f.TypeHint != "" {
		varDeclaration = "var " + f.Variable + ": " + f.TypeHint
	} else {
		varDeclaration = f.Variable
	}
	lines := []string{pad + "for " + varDeclaration + " in " + f.Iterable.ToGDScript(0) + ":"}
	if len(f.Body) == 0 {
		lines = append(lines, indent(level+1)+"pass")
	} else {
		for _, s := range f.Body {
			lines = append(lines, s.ToGDScript(level+1))
		}
	}
	return strings.Join(lines, "\n")
}

func (ForStatement) statement() {}

// MatchCase is a single case within a MatchStatement. Pattern must be either
// the wildcard string "_" (or any literal pattern string) or an Expression.
// Comment, when non-empty, appears inline after the pattern as `# comment`.
// An empty Body emits a single indented `pass`.
type MatchCase struct {
	Pattern any
	Body    []Statement
	Comment string
}

// ToGDScript renders the case header and its body at the requested indentation.
func (c MatchCase) ToGDScript(level int) string {
	pad := indent(level)
	var patternString string
	switch p := c.Pattern.(type) {
	case string:
		patternString = p
	case Expression:
		patternString = p.ToGDScript(0)
	default:
		patternString = fmt.Sprint(p)
	}
	var header string
	if c.Comment != "" {
		header = pad + patternString + ":  # " + c.Comment
	} else {
		header = pad + patternString + ":"
	}
	lines := []string{header}
	if len(c.Body) == 0 {
		lines = append(lines, indent(level+1)+"pass")
	} else {
		for _, s := range c.Body {
			lines = append(lines, s.ToGDScript(level+1))
		}
	}
	return strings.Join(lines, "\n")
}

// MatchStatement renders a GDScript `match` block with one case per entry.
type MatchStatement struct {
	Expression Expression
	Cases      []MatchCase
}

// ToGDScript renders the match expression and all of its cases.
func (m MatchStatement) ToGDScript(level int) string {
	pad := indent(level)
	lines := []string{pad + "match " + m.Expression.ToGDScript(0) + ":"}
	for _, c := range m.Cases {
		lines = append(lines, c.ToGDScript(level+1))
	}
	return strings.Join(lines, "\n")
}

func (MatchStatement) statement() {}
