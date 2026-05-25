package generator

import (
	"path/filepath"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/gdast"
)

// generateEnum converts a proto Enum into a gdast EnumDefinition.
func generateEnum(e *ast.Enum) gdast.EnumDefinition {
	values := make([]gdast.EnumValue, 0, len(e.Values))
	for _, v := range e.Values {
		number := v.Number
		values = append(values, gdast.EnumValue{Name: v.Name, Value: &number})
	}
	return gdast.EnumDefinition{Name: e.Name, Values: values}
}

// generateMessageFiles returns one GeneratedFile for the given message and one
// for each of its nested messages (recursively, flattened). classChain is the
// concatenated PascalCase parent chain (without separators) used to derive
// each generated class name; protoChain is the dotted proto-FQN-style scope
// without the package prefix, used to resolve same-file type references.
func (g *generator) generateMessageFiles(m *ast.Message, classChain, protoChain string) []GeneratedFile {
	className := g.prefix + classChain + m.Name
	scope := m.Name
	if protoChain != "" {
		scope = protoChain + "." + m.Name
	}
	class := g.generateMessageClass(m, className, scope)
	out := []GeneratedFile{{
		Filename:  className + ".pb.gd",
		ClassName: className,
		Class:     class,
		protoFQN:  scope,
	}}
	childClassChain := classChain + m.Name
	for _, nested := range m.NestedMessages {
		out = append(out, g.generateMessageFiles(nested, childClassChain, scope)...)
	}
	return out
}

// generateMessageClass produces the file-level ClassDefinition for a single
// proto message. Nested enums remain inline inside the class body; nested
// messages are emitted as sibling files by generateMessageFiles and are
// intentionally NOT included here.
func (g *generator) generateMessageClass(m *ast.Message, className, scope string) *gdast.ClassDefinition {
	prevScope := g.currentScope
	g.currentScope = scope
	defer func() { g.currentScope = prevScope }()

	var statements []gdast.Node

	for _, e := range m.NestedEnums {
		statements = append(statements,
			gdast.Comment{Text: "Nested enum"},
			generateEnum(e),
		)
	}

	statements = append(statements, gdast.Comment{Text: "Fields"})
	statements = append(statements, g.generateFieldDeclarations(m)...)

	if len(m.Oneofs) > 0 {
		statements = append(statements, gdast.Comment{Text: "Oneof enums"})
		for _, oneof := range m.Oneofs {
			statements = append(statements, generateOneofEnum(oneof))
		}
		statements = append(statements, gdast.Comment{Text: "Oneof tracking"})
		for _, oneof := range m.Oneofs {
			enumName := oneofEnumName(oneof.Name)
			statements = append(statements, gdast.VarDeclaration{
				Name:         oneofTrackingVar(oneof.Name),
				TypeHint:     enumName,
				InitialValue: gdast.RawExpression{Code: enumName + ".UNSET"},
			})
		}
	}

	statements = append(statements, gdast.Comment{Text: "Accessors"})
	statements = append(statements, g.generateAccessors(m)...)

	if len(m.Oneofs) > 0 {
		statements = append(statements, gdast.Comment{Text: "Oneof case getters"})
		for _, oneof := range m.Oneofs {
			statements = append(statements, generateOneofCaseGetter(oneof))
		}
	}

	if helpers := g.generateEnumNameAndParserHelpers(m); len(helpers) > 0 {
		statements = append(statements, gdast.Comment{Text: "Enum name lookup helpers"})
		statements = append(statements, helpers...)
	}

	statements = append(statements,
		gdast.Comment{Text: "Serialization"},
		g.generateToBytes(m),
		g.generateFromBytes(m),
		g.generateToText(m),
		g.generateFromText(m),
		g.generateToString(m),
	)

	return &gdast.ClassDefinition{
		ClassNameDirective: className,
		Extends:            "RefCounted",
		HeaderComment:      headerCommentText(filepath.Base(g.sourceName)),
		Statements:         statements,
	}
}

// generateFieldDeclarations emits the `var _name: Type = default` declarations
// for every regular, oneof, and map field of the message.
func (g *generator) generateFieldDeclarations(m *ast.Message) []gdast.Node {
	var out []gdast.Node

	for _, f := range m.Fields {
		out = append(out, g.fieldDeclaration(f))
	}

	for _, oneof := range m.Oneofs {
		for _, f := range oneof.Fields {
			out = append(out, g.fieldDeclaration(f))
		}
	}

	for _, mf := range m.Maps {
		keyType := gdscriptScalarType(mf.KeyType)
		valueType := g.renderedMapValueType(mf)
		out = append(out, gdast.VarDeclaration{
			Name:         "_" + mf.Name,
			TypeHint:     "Dictionary[" + keyType + ", " + valueType + "]",
			InitialValue: gdast.Dictionary{},
		})
	}

	return out
}

// fieldDeclaration produces a single `var _name: Type = default` declaration
// for a regular or oneof field. Repeated fields use Array[Type] = [].
func (g *generator) fieldDeclaration(f *ast.Field) gdast.VarDeclaration {
	if f.Repeated {
		gdType := g.renderedFieldType(f)
		return gdast.VarDeclaration{
			Name:         "_" + f.Name,
			TypeHint:     "Array[" + gdType + "]",
			InitialValue: gdast.Array{},
		}
	}
	gdType := g.renderedFieldType(f)
	def := g.fieldDefault(f)
	return gdast.VarDeclaration{
		Name:         "_" + f.Name,
		TypeHint:     gdType,
		InitialValue: gdast.RawExpression{Code: def},
	}
}

// fieldDefault returns the default-value expression for a field's declaration.
// Scalar fields use their proto3 zero value; enum-typed fields use the integer
// literal 0 (the proto3 enum zero value); message fields default to null.
func (g *generator) fieldDefault(f *ast.Field) string {
	if def, ok := scalarDefaultMap[f.FieldType]; ok {
		return def
	}
	if f.IsEnum {
		return "0"
	}
	return "null"
}

// gdscriptScalarType returns the GDScript type for a scalar proto type, or the
// proto type unchanged when not a scalar.
func gdscriptScalarType(protoType string) string {
	if t, ok := scalarTypeMap[protoType]; ok {
		return t
	}
	return protoType
}

// scalarTypeMap maps proto3 scalar type names to GDScript type names.
var scalarTypeMap = map[string]string{
	"double":   "float",
	"float":    "float",
	"int32":    "int",
	"int64":    "int",
	"uint32":   "int",
	"uint64":   "int",
	"sint32":   "int",
	"sint64":   "int",
	"fixed32":  "int",
	"fixed64":  "int",
	"sfixed32": "int",
	"sfixed64": "int",
	"bool":     "bool",
	"string":   "String",
	"bytes":    "PackedByteArray",
}

// scalarDefaultMap maps proto3 scalar type names to their GDScript default
// value expressions for proto3 zero semantics.
var scalarDefaultMap = map[string]string{
	"double":   "0.0",
	"float":    "0.0",
	"int32":    "0",
	"int64":    "0",
	"uint32":   "0",
	"uint64":   "0",
	"sint32":   "0",
	"sint64":   "0",
	"fixed32":  "0",
	"fixed64":  "0",
	"sfixed32": "0",
	"sfixed64": "0",
	"bool":     "false",
	"string":   `""`,
	"bytes":    "PackedByteArray()",
}
