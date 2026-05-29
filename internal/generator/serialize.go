package generator

import (
	"fmt"

	"github.com/cafecito-games/gdproto/internal/ast"
	"github.com/cafecito-games/gdproto/internal/gdast"
)

// generateToBytes builds the `to_bytes` method that serializes a message
// instance to a PackedByteArray using the proto wire format.
func (g *generator) generateToBytes(m *ast.Message) gdast.Function {
	if isEmptyMessage(m) {
		return gdast.Function{
			Name:       "to_bytes",
			ReturnType: "PackedByteArray",
			Body: []gdast.Statement{
				gdast.DocString{Text: "Serialize message to bytes."},
				gdast.Ret(gdast.Call("PackedByteArray")),
			},
		}
	}

	body := []gdast.Statement{
		gdast.DocString{Text: "Serialize message to bytes."},
		gdast.VarDeclaration{
			Name:         "result",
			TypeHint:     "PackedByteArray",
			InitialValue: gdast.Call("PackedByteArray"),
		},
	}

	for _, f := range m.Fields {
		body = append(body, g.fieldSerialization(f)...)
	}
	for _, oneof := range m.Oneofs {
		body = append(body, g.oneofSerialization(oneof)...)
	}
	for _, mf := range m.Maps {
		body = append(body, g.mapSerialization(mf)...)
	}

	body = append(body, gdast.Ret(gdast.V("result")))

	return gdast.Function{
		Name:       "to_bytes",
		ReturnType: "PackedByteArray",
		Body:       body,
	}
}

// oneofSerialization emits the `match _oneof_<group>` block that serializes
// whichever member of the oneof is currently set; the UNSET case emits no
// bytes by virtue of being absent from the match.
func (g *generator) oneofSerialization(oneof *ast.Oneof) []gdast.Statement {
	cases := make([]gdast.MatchCase, 0, len(oneof.Fields))
	for _, f := range oneof.Fields {
		tag := (f.Number << 3) | g.fieldWireType(f)
		fieldVar := "_" + f.Name
		caseBody := []gdast.Statement{
			gdast.Comment{Text: fmt.Sprintf("Field %s", f.Name)},
			rawf("result.append_array(ProtoCoreUtils.encode_varint(%d))", tag),
		}
		caseBody = append(caseBody, g.valueSerialization("result", fieldVar, f)...)
		cases = append(cases, gdast.MatchCase{
			Pattern: oneofEnumQualified(oneof.Name, f.Name),
			Body:    caseBody,
		})
	}
	return []gdast.Statement{
		gdast.Comment{Text: fmt.Sprintf("Oneof group %s", oneof.Name)},
		gdast.MatchStatement{
			Expression: gdast.V(oneofTrackingVar(oneof.Name)),
			Cases:      cases,
		},
	}
}

// fieldSerialization produces the comment and conditional/loop block that
// serializes a single regular or oneof field.
func (g *generator) fieldSerialization(f *ast.Field) []gdast.Statement {
	tag := (f.Number << 3) | g.fieldWireType(f)
	fieldVar := "_" + f.Name

	if f.Repeated {
		forBody := []gdast.Statement{
			gdast.RawStatement{Code: fmt.Sprintf("result.append_array(ProtoCoreUtils.encode_varint(%d))", tag)},
		}
		forBody = append(forBody, g.valueSerialization("result", "item", f)...)
		return []gdast.Statement{
			gdast.Comment{Text: fmt.Sprintf("Field %s (repeated)", f.Name)},
			gdast.ForStatement{
				Variable: "item",
				Iterable: gdast.V(fieldVar),
				Body:     forBody,
			},
		}
	}

	condition := g.fieldDefaultCondition(fieldVar, f)

	ifBody := []gdast.Statement{
		gdast.RawStatement{Code: fmt.Sprintf("result.append_array(ProtoCoreUtils.encode_varint(%d))", tag)},
	}
	ifBody = append(ifBody, g.valueSerialization("result", fieldVar, f)...)

	return []gdast.Statement{
		gdast.Comment{Text: fmt.Sprintf("Field %s", f.Name)},
		gdast.IfStatement{
			Condition: condition,
			Body:      ifBody,
		},
	}
}

// isEnumField reports whether f's declared type resolves to an enum.
func (g *generator) isEnumField(f *ast.Field) bool {
	return f.IsEnum
}

// fieldWireType returns the wire-type code for a field, treating enum-typed
// fields as varint per the proto3 spec. (The Python reference reproduces a
// bug here that emits wire type 2 for enums; we deliberately diverge so
// canonical protoc-generated decoders accept our output.)
func (g *generator) fieldWireType(f *ast.Field) int {
	if g.isEnumField(f) {
		return wireTypeVarint
	}
	return wireType(f.FieldType)
}

// mapValueWireType returns the wire-type code for a map value, treating
// enum-typed values as varint (same fix as fieldWireType).
func (g *generator) mapValueWireType(mf *ast.MapField) int {
	if mf.ValueIsEnum {
		return wireTypeVarint
	}
	return wireType(mf.ValueType)
}

// fieldDefaultCondition returns the GDScript expression that guards the
// serialization of a non-repeated field. Scalar types compare against their
// proto3 zero value; enum-typed fields compare against 0 (the proto3 enum
// zero value); message types compare against null.
func (g *generator) fieldDefaultCondition(fieldVar string, f *ast.Field) gdast.Expression {
	if def, ok := scalarDefaultMap[f.FieldType]; ok {
		return gdast.Ne(gdast.V(fieldVar), gdast.RawExpression{Code: def})
	}
	if g.isEnumField(f) {
		return gdast.Ne(gdast.V(fieldVar), gdast.RawExpression{Code: "0"})
	}
	return gdast.Ne(gdast.V(fieldVar), gdast.Lit(nil))
}

// valueSerialization emits the encode-call statements that append the bytes
// for a single value of the given proto field to the named target buffer.
// Target is typically "result" or "entry"; valueExpression is the GDScript
// expression evaluating to the value to encode.
func (g *generator) valueSerialization(target, valueExpression string, f *ast.Field) []gdast.Statement {
	if g.isEnumField(f) {
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_varint(%s))", target, valueExpression)}
	}
	return valueSerializationForType(target, valueExpression, f.FieldType)
}

// valueSerializationForType is the underlying type-driven serializer used for
// scalar fields and (with the enum branch handled by the caller) enum fields
// that have already been classified.
func valueSerializationForType(target, valueExpression, protoType string) []gdast.Statement {
	switch protoType {
	case "double":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_double(%s))", target, valueExpression)}
	case "float":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_float(%s))", target, valueExpression)}
	case "fixed32":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_fixed32(%s))", target, valueExpression)}
	case "sfixed32":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_sfixed32(%s))", target, valueExpression)}
	case "fixed64":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_fixed64(%s))", target, valueExpression)}
	case "sfixed64":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_sfixed64(%s))", target, valueExpression)}
	case "sint32":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_varint(ProtoCoreUtils.encode_zigzag32(%s)))", target, valueExpression)}
	case "sint64":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_varint(ProtoCoreUtils.encode_zigzag64(%s)))", target, valueExpression)}
	case "int32", "int64", "uint32", "uint64", "bool":
		return []gdast.Statement{rawf("%s.append_array(ProtoCoreUtils.encode_varint(%s))", target, valueExpression)}
	case "string":
		return []gdast.Statement{
			gdast.VarDeclaration{
				Name:         "str_data",
				TypeHint:     "PackedByteArray",
				InitialValue: gdast.RawExpression{Code: fmt.Sprintf("ProtoCoreUtils.encode_string(%s)", valueExpression)},
			},
			rawf("%s.append_array(ProtoCoreUtils.encode_varint(str_data.size()))", target),
			rawf("%s.append_array(str_data)", target),
		}
	case "bytes":
		return []gdast.Statement{
			rawf("%s.append_array(ProtoCoreUtils.encode_varint(%s.size()))", target, valueExpression),
			rawf("%s.append_array(%s)", target, valueExpression),
		}
	default:
		// Message types (and enum-typed fields, which are treated as nullable
		// references in the generated output and therefore round-trip via
		// to_bytes() like message values).
		return []gdast.Statement{
			gdast.VarDeclaration{
				Name:         "msg_data",
				TypeHint:     "PackedByteArray",
				InitialValue: gdast.RawExpression{Code: fmt.Sprintf("%s.to_bytes()", valueExpression)},
			},
			rawf("%s.append_array(ProtoCoreUtils.encode_varint(msg_data.size()))", target),
			rawf("%s.append_array(msg_data)", target),
		}
	}
}

// mapSerialization emits the comment and for-loop block that serializes a map
// field as a sequence of length-delimited entry messages.
func (g *generator) mapSerialization(mf *ast.MapField) []gdast.Statement {
	fieldVar := "_" + mf.Name
	tag := (mf.Number << 3) | wireTypeLengthDelimited
	keyTag := (1 << 3) | wireType(mf.KeyType)
	valueTag := (2 << 3) | g.mapValueWireType(mf)

	forBody := []gdast.Statement{
		typedVar("value", g.renderedMapValueType(mf), fmt.Sprintf("%s[key]", fieldVar)),
		gdast.EmptyLine{},
		gdast.Comment{Text: "Build map entry"},
		gdast.VarDeclaration{
			Name:         "entry",
			TypeHint:     "PackedByteArray",
			InitialValue: gdast.Call("PackedByteArray"),
		},
		gdast.EmptyLine{},
		gdast.Comment{Text: "Entry field 1: key"},
		rawf("entry.append_array(ProtoCoreUtils.encode_varint(%d))", keyTag),
	}
	forBody = append(forBody, mapEntryValueSerialization("key", mf.KeyType, false)...)
	forBody = append(forBody,
		gdast.EmptyLine{},
		gdast.Comment{Text: "Entry field 2: value"},
		rawf("entry.append_array(ProtoCoreUtils.encode_varint(%d))", valueTag),
	)
	forBody = append(forBody, mapEntryValueSerialization("value", mf.ValueType, mf.ValueIsEnum)...)
	forBody = append(forBody,
		gdast.EmptyLine{},
		gdast.Comment{Text: "Append entry to result"},
		rawf("result.append_array(ProtoCoreUtils.encode_varint(%d))", tag),
		rawf("result.append_array(ProtoCoreUtils.encode_varint(entry.size()))"),
		rawf("result.append_array(entry)"),
	)

	return []gdast.Statement{
		gdast.Comment{Text: fmt.Sprintf("Map field %s", mf.Name)},
		gdast.ForStatement{
			Variable: "key",
			Iterable: gdast.V(fieldVar),
			Body:     forBody,
		},
	}
}

// mapEntryValueSerialization is a parallel of valueSerialization specialized
// for map entry fields. Its variable names embed the entry-side prefix so
// that a single entry can carry distinct buffers for the key and value.
func mapEntryValueSerialization(varName, protoType string, isEnum bool) []gdast.Statement {
	if isEnum {
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_varint(%s))", varName)}
	}
	switch protoType {
	case "double":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_double(%s))", varName)}
	case "float":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_float(%s))", varName)}
	case "fixed32":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_fixed32(%s))", varName)}
	case "sfixed32":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_sfixed32(%s))", varName)}
	case "fixed64":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_fixed64(%s))", varName)}
	case "sfixed64":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_sfixed64(%s))", varName)}
	case "sint32":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_varint(ProtoCoreUtils.encode_zigzag32(%s)))", varName)}
	case "sint64":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_varint(ProtoCoreUtils.encode_zigzag64(%s)))", varName)}
	case "int32", "int64", "uint32", "uint64", "bool":
		return []gdast.Statement{rawf("entry.append_array(ProtoCoreUtils.encode_varint(%s))", varName)}
	case "string":
		return []gdast.Statement{
			gdast.VarDeclaration{
				Name:         varName + "_data",
				TypeHint:     "PackedByteArray",
				InitialValue: gdast.RawExpression{Code: fmt.Sprintf("ProtoCoreUtils.encode_string(%s)", varName)},
			},
			rawf("entry.append_array(ProtoCoreUtils.encode_varint(%s_data.size()))", varName),
			rawf("entry.append_array(%s_data)", varName),
		}
	case "bytes":
		return []gdast.Statement{
			rawf("entry.append_array(ProtoCoreUtils.encode_varint(%s.size()))", varName),
			rawf("entry.append_array(%s)", varName),
		}
	default:
		return []gdast.Statement{
			gdast.VarDeclaration{
				Name:         varName + "_msg_data",
				TypeHint:     "PackedByteArray",
				InitialValue: gdast.RawExpression{Code: fmt.Sprintf("%s.to_bytes()", varName)},
			},
			rawf("entry.append_array(ProtoCoreUtils.encode_varint(%s_msg_data.size()))", varName),
			rawf("entry.append_array(%s_msg_data)", varName),
		}
	}
}

// rawf builds a RawStatement formatted like fmt.Sprintf.
func rawf(format string, args ...any) gdast.RawStatement {
	return gdast.RawStatement{Code: fmt.Sprintf(format, args...)}
}
