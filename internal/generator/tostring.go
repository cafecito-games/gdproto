package generator

import (
	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// generateToString builds the `to_string` debug method that returns a
// human-readable representation of the message. Each non-default field is
// emitted as `name: value`, joined by commas inside `MessageName { ... }`.
// Map fields are intentionally excluded to match the reference output.
func (g *generator) generateToString(m *ast.Message) gdast.Function {
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
		Name:       "to_string",
		ReturnType: "String",
		Body:       body,
	}
}

// toStringFieldStatement renders the `if cond: parts.append(...)` block for a
// single field. Scalar and repeated fields format the value with `str(...)`;
// message-typed fields delegate to the value's own `to_string()`.
func (g *generator) toStringFieldStatement(f *ast.Field) gdast.Statement {
	fieldVar := "_" + f.Name
	var condition gdast.Expression
	var valueExpr gdast.Expression

	if f.Repeated {
		condition = gdast.BinaryOp{
			Left:  gdast.Call(fieldVar + ".size"),
			Op:    ">",
			Right: gdast.Lit(0),
		}
		valueExpr = gdast.Call("str", gdast.V(fieldVar))
	} else if def, ok := scalarDefaultMap[f.FieldType]; ok {
		condition = gdast.BinaryOp{
			Left:  gdast.V(fieldVar),
			Op:    "!=",
			Right: gdast.RawExpression{Code: def},
		}
		valueExpr = gdast.Call("str", gdast.V(fieldVar))
	} else {
		condition = gdast.BinaryOp{
			Left:  gdast.V(fieldVar),
			Op:    "!=",
			Right: gdast.Lit(nil),
		}
		valueExpr = gdast.Call(fieldVar + ".to_string")
	}

	appendCall := gdast.Call("parts.append", gdast.BinaryOp{
		Left:  gdast.Lit(f.Name + ": "),
		Op:    "+",
		Right: valueExpr,
	})

	return gdast.IfStatement{
		Condition: condition,
		Body: []gdast.Statement{
			gdast.ExpressionStatement{Expression: appendCall},
		},
	}
}
