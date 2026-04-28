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
		groups = append(groups, g.fieldAccessors(f))
	}
	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			groups = append(groups, g.oneofFieldAccessors(f, oneof))
		}
	}
	for _, mf := range m.Maps {
		groups = append(groups, g.mapAccessors(mf))
	}

	var out []gdast.Node
	for _, group := range groups {
		out = append(out, group...)
		out = append(out, gdast.EmptyLine{})
	}
	return out
}

// fieldAccessors emits the accessor functions for a regular (non-oneof) field.
func (g *generator) fieldAccessors(f *ast.Field) []gdast.Node {
	gdType := g.renderedFieldType(f)
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

	set := gdast.Function{
		Name:       "set_" + f.Name,
		Parameters: []gdast.Parameter{{Name: "value", TypeHint: gdType}},
		ReturnType: "void",
		Body:       []gdast.Statement{gdast.Assign(fieldVar, gdast.V("value"))},
	}
	get := gdast.Function{
		Name:       "get_" + f.Name,
		ReturnType: gdType,
		Body:       []gdast.Statement{gdast.Ret(gdast.V(fieldVar))},
	}
	return []gdast.Node{set, get}
}

// oneofFieldAccessors emits the setter, getter, and `has_<field>` for a field
// that participates in a oneof group. The setter clears the previously-set
// oneof field's value before updating the tracking enum and assigning the new
// value.
func (g *generator) oneofFieldAccessors(f *ast.Field, oneof *ast.Oneof) []gdast.Node {
	gdType := g.renderedFieldType(f)
	fieldVar := "_" + f.Name
	trackingVar := oneofTrackingVar(oneof.Name)
	enumValue := oneofEnumQualified(oneof.Name, f.Name)

	var clearBody []gdast.Statement
	for _, sibling := range oneof.Fields {
		if sibling.Name == f.Name {
			continue
		}
		clearBody = append(clearBody, gdast.Assign(
			"_"+sibling.Name,
			gdast.RawExpression{Code: g.fieldDefault(sibling)},
		))
	}
	clearBody = append(clearBody, gdast.Assign(trackingVar, gdast.RawExpression{Code: enumValue}))

	setBody := []gdast.Statement{
		gdast.IfStatement{
			Condition: gdast.Ne(gdast.V(trackingVar), gdast.RawExpression{Code: enumValue}),
			Body:      clearBody,
		},
		gdast.Assign(fieldVar, gdast.V("value")),
	}

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
	has := gdast.Function{
		Name:       "has_" + f.Name,
		ReturnType: "bool",
		Body: []gdast.Statement{
			gdast.Ret(gdast.Eq(gdast.V(trackingVar), gdast.RawExpression{Code: enumValue})),
		},
	}
	return []gdast.Node{set, get, has}
}

// mapAccessors emits the add/get functions for a map field. The add helper
// inserts a single key/value pair; the getter returns the underlying
// dictionary.
func (g *generator) mapAccessors(mf *ast.MapField) []gdast.Node {
	keyType := gdscriptScalarType(mf.KeyType)
	valueType := g.renderedMapValueType(mf)
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
// constructor accessor. Scalars and enum-typed fields use scalar-style
// accessors (set/get with the proto3 zero default); message-typed fields use
// the message-like accessor pair (new/get).
func (g *generator) isMessageLikeType(f *ast.Field) bool {
	if _, ok := scalarTypeMap[f.FieldType]; ok {
		return false
	}
	if f.IsEnum {
		return false
	}
	return true
}
