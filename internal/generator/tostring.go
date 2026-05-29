package generator

import (
	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/gdast"
)

// generateToString builds the `_to_string` debug method that returns a
// human-readable representation of the message in the form
// `MessageName { field: value, ... }`. Fields equal to their proto3 default
// are skipped. Regular fields are emitted in source order, followed by map
// fields, followed by oneof fields.
func (g *generator) generateToString(m *ast.Message) gdast.Function {
	if isEmptyMessage(m) {
		return gdast.Function{
			Name:       "_to_string",
			ReturnType: "String",
			Body: []gdast.Statement{
				gdast.DocString{Text: "Generate debug string representation."},
				gdast.Ret(gdast.Lit(m.Name + " {}")),
			},
		}
	}

	body := []gdast.Statement{
		gdast.DocString{Text: "Generate debug string representation."},
		gdast.VarDeclaration{
			Name:         "parts",
			TypeHint:     "Array[String]",
			InitialValue: gdast.Array{},
		},
		gdast.EmptyLine{},
	}

	for _, f := range m.Fields {
		body = append(body, g.toStringFieldStatement(f))
	}
	for _, mf := range m.Maps {
		body = append(body, toStringMapStatement(mf))
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			body = append(body, g.toStringFieldStatement(f))
		}
	}

	body = append(body,
		gdast.EmptyLine{},
		gdast.Ret(gdast.BinaryOp{
			Left: gdast.BinaryOp{
				Left:  gdast.Lit(m.Name + " { "),
				Op:    "+",
				Right: gdast.Call(`", ".join`, gdast.V("parts")),
			},
			Op:    "+",
			Right: gdast.Lit(" }"),
		}),
	)

	return gdast.Function{
		Name:       "_to_string",
		ReturnType: "String",
		Body:       body,
	}
}

// toStringFieldStatement renders the `if cond: parts.append(...)` block for a
// single non-map field. Scalar, enum, repeated, and message-typed fields all
// format the value with `str(...)`. Repeated fields gate on `.size() > 0`,
// message fields gate on `!= null`, and scalar/enum fields gate on the
// proto3 zero value.
func (g *generator) toStringFieldStatement(f *ast.Field) gdast.Statement {
	fieldVar := "_" + f.Name
	var condition gdast.Expression

	switch {
	case f.Repeated:
		condition = gdast.BinaryOp{
			Left:  gdast.Call(fieldVar + ".size"),
			Op:    ">",
			Right: gdast.Lit(0),
		}
	case f.IsEnum:
		condition = gdast.BinaryOp{
			Left:  gdast.V(fieldVar),
			Op:    "!=",
			Right: gdast.RawExpression{Code: "0"},
		}
	default:
		if def, ok := scalarDefaultMap[f.FieldType]; ok {
			condition = gdast.BinaryOp{
				Left:  gdast.V(fieldVar),
				Op:    "!=",
				Right: gdast.RawExpression{Code: def},
			}
		} else {
			condition = gdast.BinaryOp{
				Left:  gdast.V(fieldVar),
				Op:    "!=",
				Right: gdast.Lit(nil),
			}
		}
	}

	appendCall := gdast.Call("parts.append", gdast.BinaryOp{
		Left:  gdast.Lit(f.Name + ": "),
		Op:    "+",
		Right: gdast.Call("str", gdast.V(fieldVar)),
	})

	return gdast.IfStatement{
		Condition: condition,
		Body: []gdast.Statement{
			gdast.ExpressionStatement{Expression: appendCall},
		},
	}
}

// toStringMapStatement renders the `if _name.size() > 0: parts.append(...)`
// block for a map field.
func toStringMapStatement(mf *ast.MapField) gdast.Statement {
	fieldVar := "_" + mf.Name
	condition := gdast.BinaryOp{
		Left:  gdast.Call(fieldVar + ".size"),
		Op:    ">",
		Right: gdast.Lit(0),
	}
	appendCall := gdast.Call("parts.append", gdast.BinaryOp{
		Left:  gdast.Lit(mf.Name + ": "),
		Op:    "+",
		Right: gdast.Call("str", gdast.V(fieldVar)),
	})
	return gdast.IfStatement{
		Condition: condition,
		Body: []gdast.Statement{
			gdast.ExpressionStatement{Expression: appendCall},
		},
	}
}
