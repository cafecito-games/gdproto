package gdast

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RawExpression embeds an opaque snippet of GDScript source in expression
// position. Multi-line snippets have non-blank lines prefixed with the
// requested indentation; blank lines remain empty.
type RawExpression struct {
	Code string
}

// ToGDScript prefixes every non-blank line of Code with the requested
// indentation, preserving relative tabs already present in the snippet.
func (r RawExpression) ToGDScript(level int) string {
	if level == 0 {
		return r.Code
	}
	if !strings.Contains(r.Code, "\n") {
		return indent(level) + r.Code
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

func (RawExpression) expression() {}

// Literal is a literal value with automatic GDScript formatting. It handles
// nil -> "null", bool -> "true"/"false", smart-quoted strings, INF/NAN for
// special floats, and standard numeric formatting.
//
// Float formatting note: Go's strconv.FormatFloat(v, 'g', -1, 64) is used for
// regular float64 values. This matches Python's str(f) for typical values
// (e.g. 3.14) but may diverge for large magnitudes (e.g. 1.5e10 produces
// "1.5e+10" in Go vs. "15000000000.0" in Python). The common proto-default
// cases round-trip identically.
type Literal struct {
	Value any
}

// ToGDScript formats the underlying value as GDScript source.
func (l Literal) ToGDScript(_ int) string {
	switch v := l.Value.(type) {
	case nil:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		hasDouble := strings.ContainsRune(v, '"')
		hasSingle := strings.ContainsRune(v, '\'')
		if hasDouble && !hasSingle {
			return "'" + v + "'"
		}
		if hasDouble && hasSingle {
			escaped := strings.ReplaceAll(v, `\`, `\\`)
			escaped = strings.ReplaceAll(escaped, `"`, `\"`)
			return `"` + escaped + `"`
		}
		return `"` + v + `"`
	case float64:
		return formatFloat(v)
	case float32:
		return formatFloat(float64(v))
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return fmt.Sprint(v)
	}
}

func (Literal) expression() {}

func formatFloat(v float64) string {
	if math.IsInf(v, 1) {
		return "INF"
	}
	if math.IsInf(v, -1) {
		return "-INF"
	}
	if math.IsNaN(v) {
		return "NAN"
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// Variable is a variable reference by name.
type Variable struct {
	Name string
}

// ToGDScript returns the variable name unchanged.
func (v Variable) ToGDScript(_ int) string { return v.Name }

func (Variable) expression() {}

// BinaryOp is a binary operation joining two expressions with an operator.
type BinaryOp struct {
	Left  Expression
	Op    string
	Right Expression
}

// ToGDScript renders the operands separated by the operator with single spaces.
func (b BinaryOp) ToGDScript(_ int) string {
	return b.Left.ToGDScript(0) + " " + b.Op + " " + b.Right.ToGDScript(0)
}

func (BinaryOp) expression() {}

// UnaryOp is a unary operation. Word operators ("not", "await") are followed
// by a space; symbolic operators are not.
type UnaryOp struct {
	Op      string
	Operand Expression
}

// ToGDScript renders the operator and operand with the appropriate spacing.
func (u UnaryOp) ToGDScript(_ int) string {
	if u.Op == "not" || u.Op == "await" {
		return u.Op + " " + u.Operand.ToGDScript(0)
	}
	return u.Op + u.Operand.ToGDScript(0)
}

func (UnaryOp) expression() {}

// CallExpr is a function or method call. Function may be a string (bare name)
// or an Expression (for method chains and computed callees).
type CallExpr struct {
	Function  any
	Arguments []Expression
}

// ToGDScript renders the call as `function(arg1, arg2, ...)`.
func (c CallExpr) ToGDScript(_ int) string {
	var functionString string
	switch f := c.Function.(type) {
	case string:
		functionString = f
	case Expression:
		functionString = f.ToGDScript(0)
	default:
		functionString = fmt.Sprint(f)
	}
	parts := make([]string, len(c.Arguments))
	for i, argument := range c.Arguments {
		parts[i] = argument.ToGDScript(0)
	}
	return functionString + "(" + strings.Join(parts, ", ") + ")"
}

func (CallExpr) expression() {}

// GetAttr is an attribute access via dot notation.
type GetAttr struct {
	Object    Expression
	Attribute string
}

// ToGDScript renders the access as `object.attribute`.
func (g GetAttr) ToGDScript(_ int) string {
	return g.Object.ToGDScript(0) + "." + g.Attribute
}

func (GetAttr) expression() {}

// Subscript is a subscript access via brackets.
type Subscript struct {
	Object Expression
	Key    Expression
}

// ToGDScript renders the access as `object[key]`.
func (s Subscript) ToGDScript(_ int) string {
	return s.Object.ToGDScript(0) + "[" + s.Key.ToGDScript(0) + "]"
}

func (Subscript) expression() {}

// Array is an array literal rendered inline.
type Array struct {
	Elements []Expression
}

// ToGDScript renders the elements as `[a, b, c]`, or `[]` when empty.
func (a Array) ToGDScript(_ int) string {
	if len(a.Elements) == 0 {
		return "[]"
	}
	parts := make([]string, len(a.Elements))
	for i, element := range a.Elements {
		parts[i] = element.ToGDScript(0)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (Array) expression() {}

// DictPair is a single key/value pair within a Dictionary literal.
type DictPair struct {
	Key   Expression
	Value Expression
}

// Dictionary is a dictionary literal rendered inline.
type Dictionary struct {
	Pairs []DictPair
}

// ToGDScript renders the pairs as `{k: v, ...}`, or `{}` when empty.
func (d Dictionary) ToGDScript(_ int) string {
	if len(d.Pairs) == 0 {
		return "{}"
	}
	parts := make([]string, len(d.Pairs))
	for i, pair := range d.Pairs {
		parts[i] = pair.Key.ToGDScript(0) + ": " + pair.Value.ToGDScript(0)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (Dictionary) expression() {}

// TypeCast is a type cast using the `as` keyword.
type TypeCast struct {
	Value    Expression
	TypeName string
}

// ToGDScript renders the cast as `value as TypeName`.
func (t TypeCast) ToGDScript(_ int) string {
	return t.Value.ToGDScript(0) + " as " + t.TypeName
}

func (TypeCast) expression() {}

// TernaryOp is a ternary conditional expression rendered as
// `true_value if condition else false_value`.
type TernaryOp struct {
	Condition  Expression
	TrueValue  Expression
	FalseValue Expression
}

// ToGDScript renders the ternary in GDScript order.
func (t TernaryOp) ToGDScript(_ int) string {
	return t.TrueValue.ToGDScript(0) + " if " + t.Condition.ToGDScript(0) + " else " + t.FalseValue.ToGDScript(0)
}

func (TernaryOp) expression() {}
