package generator

import (
	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// generateAccessors produces setter, getter, and oneof presence-check methods
// for every field of a message. The bodies emitted here are minimal but
// functional; later tasks may extend them with validation or change-tracking.
func (g *generator) generateAccessors(m *ast.Message) []gdast.Node {
	var groups [][]gdast.Node

	for _, f := range m.Fields {
		groups = append(groups, g.fieldAccessors(f, ""))
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			groups = append(groups, g.fieldAccessors(f, oneof.Name))
		}
	}
	for _, mf := range m.Maps {
		groups = append(groups, g.mapAccessors(mf))
	}

	var out []gdast.Node
	for i, group := range groups {
		if i > 0 {
			out = append(out, gdast.EmptyLine{})
		}
		out = append(out, group...)
	}
	return out
}

// fieldAccessors emits the accessor functions for a regular or oneof field.
// When oneofName is non-empty the field is part of a oneof and its setter
// updates the parent's oneof tracking variable; a presence check is also
// emitted.
func (g *generator) fieldAccessors(f *ast.Field, oneofName string) []gdast.Node {
	gdType := g.typeName(f.FieldType)
	fieldVar := "_" + f.Name

	if f.Repeated {
		var add gdast.Function
		if g.isMessageLikeType(f) {
			add = gdast.Function{
				Name:       "add_" + f.Name,
				ReturnType: gdType,
				Body: []gdast.Statement{
					gdast.VarDeclaration{
						Name:         "item",
						TypeHint:     gdType,
						InitialValue: gdast.Call(gdast.V(gdType + ".new")),
					},
					gdast.ExpressionStatement{Expression: gdast.Call(
						gdast.Attr(gdast.V(fieldVar), "append"),
						gdast.V("item"),
					)},
					gdast.Ret(gdast.V("item")),
				},
			}
		} else {
			add = gdast.Function{
				Name:       "add_" + f.Name,
				Parameters: []gdast.Parameter{{Name: "value", TypeHint: gdType}},
				ReturnType: "void",
				Body: []gdast.Statement{
					gdast.ExpressionStatement{Expression: gdast.Call(
						gdast.Attr(gdast.V(fieldVar), "append"),
						gdast.V("value"),
					)},
				},
			}
		}
		get := gdast.Function{
			Name:       "get_" + f.Name,
			ReturnType: "Array[" + gdType + "]",
			Body:       []gdast.Statement{gdast.Ret(gdast.V(fieldVar))},
		}
		return []gdast.Node{add, get}
	}

	if g.isMessageLikeType(f) {
		newer := gdast.Function{
			Name:       "new_" + f.Name,
			ReturnType: gdType,
			Body: []gdast.Statement{
				gdast.Assign(fieldVar, gdast.Call(gdType+".new")),
				gdast.Ret(gdast.V(fieldVar)),
			},
		}
		get := gdast.Function{
			Name:       "get_" + f.Name,
			ReturnType: gdType,
			Body:       []gdast.Statement{gdast.Ret(gdast.V(fieldVar))},
		}
		return []gdast.Node{newer, get}
	}

	var setBody []gdast.Statement
	if oneofName != "" {
		oneofVar := "_oneof_" + oneofName
		setBody = append(setBody, gdast.IfStatement{
			Condition: gdast.Ne(gdast.V(oneofVar), gdast.Lit(f.Name)),
			Body: []gdast.Statement{
				gdast.Assign(oneofVar, gdast.Lit(f.Name)),
			},
		})
	}
	setBody = append(setBody, gdast.Assign(fieldVar, gdast.V("value")))
	set := gdast.Function{
		Name:       "set_" + f.Name,
		Parameters: []gdast.Parameter{{Name: "value", TypeHint: gdType}},
		ReturnType: "void",
		Body:       setBody,
	}
	get := gdast.Function{
		Name:       "get_" + f.Name,
		ReturnType: gdType,
		Body:       []gdast.Statement{gdast.Ret(gdast.V(fieldVar))},
	}
	out := []gdast.Node{set, get}
	if oneofName != "" {
		has := gdast.Function{
			Name:       "has_" + f.Name,
			ReturnType: "bool",
			Body: []gdast.Statement{
				gdast.Ret(gdast.Eq(gdast.V("_oneof_"+oneofName), gdast.Lit(f.Name))),
			},
		}
		out = append(out, has)
	}
	return out
}

// mapAccessors emits the add/get functions for a map field. The add helper
// inserts a single key/value pair; the getter returns the underlying
// dictionary.
func (g *generator) mapAccessors(mf *ast.MapField) []gdast.Node {
	keyType := gdscriptScalarType(mf.KeyType)
	valueType := g.typeName(mf.ValueType)
	fieldVar := "_" + mf.Name

	add := gdast.Function{
		Name: "add_" + mf.Name,
		Parameters: []gdast.Parameter{
			{Name: "key", TypeHint: keyType},
			{Name: "value", TypeHint: valueType},
		},
		ReturnType: "void",
		Body: []gdast.Statement{
			gdast.Assign(
				gdast.Subscript{Object: gdast.V(fieldVar), Key: gdast.V("key")},
				gdast.V("value"),
			),
		},
	}
	get := gdast.Function{
		Name:       "get_" + mf.Name,
		ReturnType: "Dictionary[" + keyType + ", " + valueType + "]",
		Body:       []gdast.Statement{gdast.Ret(gdast.V(fieldVar))},
	}
	return []gdast.Node{add, get}
}

// isMessageLikeType reports whether the field's type renders with a `new_`
// constructor accessor. Scalars and enums imported from another proto file
// use scalar-style accessors. Same-file enums fall through to the
// message-like branch to preserve compatibility with the reference output,
// which models locally-defined enums as null-defaulting references.
func (g *generator) isMessageLikeType(f *ast.Field) bool {
	if _, ok := scalarTypeMap[f.FieldType]; ok {
		return false
	}
	if f.IsEnum && f.SourceFile != "" && f.SourceFile != g.sourceName {
		return false
	}
	return true
}
