package generator

import (
	"fmt"
	"strconv"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// inferredVar emits `var name := <code>` (walrus form) as a RawStatement,
// matching the reference generator's local-variable declarations inside
// from_bytes/decode helpers.
func inferredVar(name, code string) gdast.RawStatement {
	return gdast.RawStatement{Code: "var " + name + " := " + code}
}

// generateFromBytes builds the `from_bytes` method that deserializes a
// PackedByteArray into the message's fields. The structure mirrors the
// reference Python generator: a while-loop reads tag/wire-type pairs, then
// dispatches on field number via a match statement; each case implements the
// per-field decode for its wire format.
func (g *generator) generateFromBytes(m *ast.Message) gdast.Function {
	body := []gdast.Statement{
		gdast.DocString{Text: "Deserialize message from bytes."},
		gdast.VarDeclaration{
			Name:         "offset",
			TypeHint:     "int",
			InitialValue: gdast.Lit(0),
		},
		gdast.EmptyLine{},
	}

	whileBody := tagReadingStatements()

	totalCases := len(m.Fields) + len(m.Maps) + 1
	for _, o := range m.Oneofs {
		totalCases += len(o.Fields)
	}
	cases := make([]gdast.MatchCase, 0, totalCases)
	for _, f := range m.Fields {
		cases = append(cases, gdast.MatchCase{
			Pattern: strconv.Itoa(f.Number),
			Body:    g.fieldDeserialization(f),
		})
	}
	for _, mf := range m.Maps {
		caseBody := []gdast.Statement{
			gdast.Comment{Text: fmt.Sprintf("Map field %s", mf.Name)},
		}
		caseBody = append(caseBody, mapFieldDeserialization(mf)...)
		cases = append(cases, gdast.MatchCase{
			Pattern: strconv.Itoa(mf.Number),
			Body:    caseBody,
		})
	}
	for _, o := range m.Oneofs {
		for _, f := range o.Fields {
			body := g.fieldDeserialization(f)
			body = append(body, rawf("%s = %s",
				oneofTrackingVar(o.Name),
				oneofEnumQualified(o.Name, f.Name),
			))
			cases = append(cases, gdast.MatchCase{
				Pattern: strconv.Itoa(f.Number),
				Body:    body,
			})
		}
	}
	cases = append(cases, gdast.MatchCase{
		Pattern: "_",
		Body: []gdast.Statement{
			gdast.Comment{Text: "Skip unknown field"},
			skipUnknownFieldMatch(),
		},
	})

	whileBody = append(whileBody, gdast.MatchStatement{
		Expression: gdast.V("field_number"),
		Cases:      cases,
	})

	body = append(body,
		gdast.WhileStatement{
			Condition: gdast.Lt(gdast.V("offset"), gdast.RawExpression{Code: "data.size()"}),
			Body:      whileBody,
		},
		gdast.EmptyLine{},
		gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.NO_ERRORS"}),
	)

	return gdast.Function{
		Name:       "from_bytes",
		Parameters: []gdast.Parameter{{Name: "data", TypeHint: "PackedByteArray"}},
		ReturnType: "ProtoCoreUtils.ProtobufError",
		Body:       body,
	}
}

// tagReadingStatements emits the prelude inside the while-loop that reads a
// single field tag and decodes it into `field_number` and `wire_type`.
func tagReadingStatements() []gdast.Statement {
	return []gdast.Statement{
		gdast.Comment{Text: "Read field tag"},
		inferredVar("tag_result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "tag_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
		gdast.VarDeclaration{
			Name:         "tag",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "tag_result.value"},
		},
		rawf("offset += tag_result.size"),
		gdast.EmptyLine{},
		gdast.VarDeclaration{
			Name:         "field_number",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "ProtoCoreUtils.get_field_number(tag)"},
		},
		gdast.VarDeclaration{
			Name:         "wire_type",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "ProtoCoreUtils.get_wire_type(tag)"},
		},
		gdast.EmptyLine{},
	}
}

// skipUnknownFieldMatch builds the nested match-on-wire-type block that
// advances `offset` past an unknown field's payload.
func skipUnknownFieldMatch() gdast.MatchStatement {
	return gdast.MatchStatement{
		Expression: gdast.V("wire_type"),
		Cases: []gdast.MatchCase{
			{
				Pattern: "0",
				Comment: "Varint",
				Body: []gdast.Statement{
					inferredVar("skip_result", "ProtoCoreUtils.decode_varint(data, offset)"),
					gdast.IfStatement{
						Condition: gdast.Eq(gdast.RawExpression{Code: "skip_result.size"}, gdast.Lit(-1)),
						Body: []gdast.Statement{
							gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
						},
					},
					rawf("offset += skip_result.size"),
				},
			},
			{
				Pattern: "1",
				Comment: "Fixed64",
				Body:    []gdast.Statement{rawf("offset += 8")},
			},
			{
				Pattern: "2",
				Comment: "Length-delimited",
				Body: []gdast.Statement{
					inferredVar("skip_length_result", "ProtoCoreUtils.decode_varint(data, offset)"),
					gdast.IfStatement{
						Condition: gdast.Eq(gdast.RawExpression{Code: "skip_length_result.size"}, gdast.Lit(-1)),
						Body: []gdast.Statement{
							gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND"}),
						},
					},
					rawf("offset += skip_length_result.size + skip_length_result.value"),
				},
			},
			{
				Pattern: "5",
				Comment: "Fixed32",
				Body:    []gdast.Statement{rawf("offset += 4")},
			},
			{
				Pattern: "_",
				Body: []gdast.Statement{
					gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.UNDEFINED_STATE"}),
				},
			},
		},
	}
}

// fieldDeserialization emits the body of a single match case, including the
// leading comment, for a regular or repeated field.
func (g *generator) fieldDeserialization(f *ast.Field) []gdast.Statement {
	out := []gdast.Statement{gdast.Comment{Text: fmt.Sprintf("Field %s", f.Name)}}
	if f.Repeated {
		out = append(out, g.repeatedFieldDeserialization(f)...)
	} else {
		out = append(out, g.singleFieldDeserialization(f)...)
	}
	return out
}

// singleFieldDeserialization decodes a non-repeated scalar/message/enum field
// inline into its private storage variable, then advances `offset`.
func (g *generator) singleFieldDeserialization(f *ast.Field) []gdast.Statement {
	fieldVar := "_" + f.Name
	if g.isEnumField(f) {
		return varintAssign(fieldVar, false)
	}
	switch f.FieldType {
	case "int32", "int64", "uint32", "uint64", "bool":
		return varintAssign(fieldVar, f.FieldType == "bool")
	case "sint32":
		return zigzagAssign(fieldVar, "ProtoCoreUtils.decode_zigzag32")
	case "sint64":
		return zigzagAssign(fieldVar, "ProtoCoreUtils.decode_zigzag64")
	case "float":
		return fixedAssign(fieldVar, "ProtoCoreUtils.decode_float", 4)
	case "double":
		return fixedAssign(fieldVar, "ProtoCoreUtils.decode_double", 8)
	case "fixed32":
		return fixedAssign(fieldVar, "ProtoCoreUtils.decode_fixed32", 4)
	case "sfixed32":
		return fixedAssign(fieldVar, "ProtoCoreUtils.decode_sfixed32", 4)
	case "fixed64":
		return fixedAssign(fieldVar, "ProtoCoreUtils.decode_fixed64", 8)
	case "sfixed64":
		return fixedAssign(fieldVar, "ProtoCoreUtils.decode_sfixed64", 8)
	case "string":
		return stringAssign(fieldVar)
	case "bytes":
		return bytesAssign(fieldVar)
	default:
		// Treated as a message type (also covers enum-typed nullable refs).
		typeName := g.typeName(f.FieldType)
		return messageAssign(fieldVar, typeName)
	}
}

// repeatedFieldDeserialization decodes a repeated field. Numeric (varint /
// fixed) types must accept either packed or unpacked encodings; string,
// bytes, and message types are always length-delimited per element.
func (g *generator) repeatedFieldDeserialization(f *ast.Field) []gdast.Statement {
	fieldVar := "_" + f.Name
	switch f.FieldType {
	case "string":
		return repeatedStringAppend(fieldVar)
	case "bytes":
		return repeatedBytesAppend(fieldVar)
	default:
		if _, ok := scalarTypeMap[f.FieldType]; ok {
			// Numeric scalar - covers varint, zigzag, fixed widths.
			return repeatedNumericPackedOrUnpacked(fieldVar, f.FieldType)
		}
		typeName := g.typeName(f.FieldType)
		return repeatedMessageAppend(fieldVar, typeName)
	}
}

// varintAssign decodes a varint into fieldVar. When asBool, the assignment
// converts the integer to a bool via `!= 0`.
func varintAssign(fieldVar string, asBool bool) []gdast.Statement {
	stmts := []gdast.Statement{
		inferredVar("result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
	}
	if asBool {
		stmts = append(stmts, rawf("%s = result.value != 0", fieldVar))
	} else {
		stmts = append(stmts, rawf("%s = result.value", fieldVar))
	}
	stmts = append(stmts, rawf("offset += result.size"))
	return stmts
}

// zigzagAssign decodes a varint then applies a zig-zag transform to recover
// the signed value before assigning into fieldVar.
func zigzagAssign(fieldVar, decodeFunc string) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
		rawf("%s = %s(result.value)", fieldVar, decodeFunc),
		rawf("offset += result.size"),
	}
}

// fixedAssign decodes a fixed-width (4 or 8 byte) value into fieldVar after
// bounds-checking the remaining buffer.
func fixedAssign(fieldVar, decodeFunc string, width int) []gdast.Statement {
	return []gdast.Statement{
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: fmt.Sprintf("offset + %d > data.size()", width)},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.PARSE_INCOMPLETE"}),
			},
		},
		rawf("%s = %s(data, offset)", fieldVar, decodeFunc),
		rawf("offset += %d", width),
	}
}

// stringAssign decodes a length-prefixed UTF-8 string into fieldVar.
func stringAssign(fieldVar string) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		rawf("%s = ProtoCoreUtils.decode_string(data, offset, length)", fieldVar),
		rawf("offset += length"),
	)
	return stmts
}

// bytesAssign decodes a length-prefixed raw byte slice into fieldVar.
func bytesAssign(fieldVar string) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		rawf("%s = data.slice(offset, offset + length)", fieldVar),
		rawf("offset += length"),
	)
	return stmts
}

// messageAssign decodes a length-prefixed embedded message (or enum-typed
// reference) into fieldVar by constructing a new instance and recursing into
// its from_bytes; failures propagate the inner error code.
func messageAssign(fieldVar, typeName string) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		gdast.VarDeclaration{
			Name:         "msg_data",
			TypeHint:     "PackedByteArray",
			InitialValue: gdast.RawExpression{Code: "data.slice(offset, offset + length)"},
		},
		rawf("%s = %s.new()", fieldVar, typeName),
		inferredVar("msg_result", fmt.Sprintf("%s.from_bytes(msg_data)", fieldVar)),
		gdast.IfStatement{
			Condition: gdast.Ne(gdast.V("msg_result"), gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.NO_ERRORS"}),
			Body: []gdast.Statement{
				gdast.Ret(gdast.V("msg_result")),
			},
		},
		rawf("offset += length"),
	)
	return stmts
}

// lengthPrefixDecode emits the standard preamble that decodes a varint length
// prefix and validates it against the remaining buffer size.
func lengthPrefixDecode() []gdast.Statement {
	return []gdast.Statement{
		inferredVar("length_result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "length_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND"}),
			},
		},
		rawf("offset += length_result.size"),
		gdast.VarDeclaration{
			Name:         "length",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "length_result.value"},
		},
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: "offset + length > data.size()"},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"}),
			},
		},
	}
}

// repeatedStringAppend appends a single decoded string to a repeated field's
// backing array. Strings are always length-delimited so there is no packed
// form to consider.
func repeatedStringAppend(fieldVar string) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		rawf("%s.append(ProtoCoreUtils.decode_string(data, offset, length))", fieldVar),
		rawf("offset += length"),
	)
	return stmts
}

// repeatedBytesAppend appends a single decoded byte slice to a repeated
// field's backing array.
func repeatedBytesAppend(fieldVar string) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		rawf("%s.append(data.slice(offset, offset + length))", fieldVar),
		rawf("offset += length"),
	)
	return stmts
}

// repeatedMessageAppend decodes one embedded message and appends it to the
// repeated field's backing array, propagating decode errors from the nested
// message verbatim.
func repeatedMessageAppend(fieldVar, typeName string) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		gdast.VarDeclaration{
			Name:         "msg_data",
			TypeHint:     "PackedByteArray",
			InitialValue: gdast.RawExpression{Code: "data.slice(offset, offset + length)"},
		},
		inferredVar("msg_item", fmt.Sprintf("%s.new()", typeName)),
		inferredVar("msg_result", "msg_item.from_bytes(msg_data)"),
		gdast.IfStatement{
			Condition: gdast.Ne(gdast.V("msg_result"), gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.NO_ERRORS"}),
			Body: []gdast.Statement{
				gdast.Ret(gdast.V("msg_result")),
			},
		},
		rawf("%s.append(msg_item)", fieldVar),
		rawf("offset += length"),
	)
	return stmts
}

// repeatedNumericPackedOrUnpacked handles a repeated numeric field by
// branching on the wire type at runtime: a length-delimited tag means a
// packed payload (a nested while-loop drains it), and any other wire type
// means a single unpacked value matching the element type.
func repeatedNumericPackedOrUnpacked(fieldVar, protoType string) []gdast.Statement {
	packedBody := []gdast.Statement{
		inferredVar("length_result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "length_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND"}),
			},
		},
		rawf("offset += length_result.size"),
		gdast.VarDeclaration{
			Name:         "length",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "length_result.value"},
		},
		gdast.VarDeclaration{
			Name:         "end_offset",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "offset + length"},
		},
		gdast.WhileStatement{
			Condition: gdast.Lt(gdast.V("offset"), gdast.V("end_offset")),
			Body:      packedElementDecode(fieldVar, protoType),
		},
	}
	unpackedBody := singleNumericAppend(fieldVar, protoType)
	return []gdast.Statement{
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.V("wire_type"), gdast.Lit(2)),
			Body:      packedBody,
			ElseBody:  unpackedBody,
		},
	}
}

// packedElementDecode emits the body of the inner while-loop that drains a
// packed numeric repeated field, appending each element to fieldVar.
func packedElementDecode(fieldVar, protoType string) []gdast.Statement {
	switch protoType {
	case "int32", "int64", "uint32", "uint64":
		return varintAppend(fieldVar, false)
	case "bool":
		return varintAppend(fieldVar, true)
	case "sint32":
		return zigzagAppend(fieldVar, "ProtoCoreUtils.decode_zigzag32")
	case "sint64":
		return zigzagAppend(fieldVar, "ProtoCoreUtils.decode_zigzag64")
	case "float":
		return fixedAppend(fieldVar, "ProtoCoreUtils.decode_float", 4)
	case "double":
		return fixedAppend(fieldVar, "ProtoCoreUtils.decode_double", 8)
	case "fixed32":
		return fixedAppend(fieldVar, "ProtoCoreUtils.decode_fixed32", 4)
	case "sfixed32":
		return fixedAppend(fieldVar, "ProtoCoreUtils.decode_sfixed32", 4)
	case "fixed64":
		return fixedAppend(fieldVar, "ProtoCoreUtils.decode_fixed64", 8)
	case "sfixed64":
		return fixedAppend(fieldVar, "ProtoCoreUtils.decode_sfixed64", 8)
	}
	return nil
}

// singleNumericAppend decodes one numeric value of the given protoType and
// appends it to fieldVar; used for the unpacked branch of a repeated field.
func singleNumericAppend(fieldVar, protoType string) []gdast.Statement {
	return packedElementDecode(fieldVar, protoType)
}

// varintAppend decodes a single varint and appends it to fieldVar. When
// asBool, the value is converted with `!= 0`.
func varintAppend(fieldVar string, asBool bool) []gdast.Statement {
	stmts := []gdast.Statement{
		inferredVar("result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
	}
	if asBool {
		stmts = append(stmts, rawf("%s.append(result.value != 0)", fieldVar))
	} else {
		stmts = append(stmts, rawf("%s.append(result.value)", fieldVar))
	}
	stmts = append(stmts, rawf("offset += result.size"))
	return stmts
}

// zigzagAppend decodes a varint, applies the zig-zag transform, and appends
// the result to fieldVar.
func zigzagAppend(fieldVar, decodeFunc string) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("result", "ProtoCoreUtils.decode_varint(data, offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
		rawf("%s.append(%s(result.value))", fieldVar, decodeFunc),
		rawf("offset += result.size"),
	}
}

// fixedAppend decodes a fixed-width numeric value and appends it to fieldVar.
func fixedAppend(fieldVar, decodeFunc string, width int) []gdast.Statement {
	return []gdast.Statement{
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: fmt.Sprintf("offset + %d > data.size()", width)},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.PARSE_INCOMPLETE"}),
			},
		},
		rawf("%s.append(%s(data, offset))", fieldVar, decodeFunc),
		rawf("offset += %d", width),
	}
}

// mapFieldDeserialization decodes a single map<K,V> entry message: it reads
// the length-prefixed entry, then iterates the entry's tag stream, decoding
// each `1: key` / `2: value` pair before storing it into the map field.
func mapFieldDeserialization(mf *ast.MapField) []gdast.Statement {
	stmts := lengthPrefixDecode()
	stmts = append(stmts,
		gdast.EmptyLine{},
		gdast.VarDeclaration{
			Name:         "entry_data",
			TypeHint:     "PackedByteArray",
			InitialValue: gdast.RawExpression{Code: "data.slice(offset, offset + length)"},
		},
		gdast.VarDeclaration{
			Name:         "entry_offset",
			TypeHint:     "int",
			InitialValue: gdast.Lit(0),
		},
		gdast.EmptyLine{},
		inferredVar("map_key", mapDefault(mf.KeyType)),
		inferredVar("map_value", mapDefault(mf.ValueType)),
		gdast.EmptyLine{},
		gdast.WhileStatement{
			Condition: gdast.Lt(gdast.V("entry_offset"), gdast.RawExpression{Code: "entry_data.size()"}),
			Body:      mapEntryLoopBody(mf),
		},
		gdast.EmptyLine{},
		rawf("_%s[map_key] = map_value", mf.Name),
		rawf("offset += length"),
	)
	return stmts
}

// mapEntryLoopBody emits the inner while-loop body that reads one entry tag
// from `entry_data` and dispatches to either the key or value decoder.
func mapEntryLoopBody(mf *ast.MapField) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("entry_tag_result", "ProtoCoreUtils.decode_varint(entry_data, entry_offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "entry_tag_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
		gdast.VarDeclaration{
			Name:         "entry_tag",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "entry_tag_result.value"},
		},
		rawf("entry_offset += entry_tag_result.size"),
		gdast.VarDeclaration{
			Name:         "entry_field_number",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "ProtoCoreUtils.get_field_number(entry_tag)"},
		},
		gdast.EmptyLine{},
		gdast.MatchStatement{
			Expression: gdast.V("entry_field_number"),
			Cases: []gdast.MatchCase{
				{
					Pattern: "1",
					Body: append(
						[]gdast.Statement{gdast.Comment{Text: "Entry key"}},
						mapEntryDecode("map_key", mf.KeyType)...,
					),
				},
				{
					Pattern: "2",
					Body: append(
						[]gdast.Statement{gdast.Comment{Text: "Entry value"}},
						mapEntryDecode("map_value", mf.ValueType)...,
					),
				},
			},
		},
	}
}

// mapEntryDecode produces the inline decode for a single map-entry field,
// reading from `entry_data`/`entry_offset` and assigning into target.
func mapEntryDecode(target, protoType string) []gdast.Statement {
	switch protoType {
	case "int32", "int64", "uint32", "uint64", "bool":
		return mapVarintAssign(target, protoType == "bool")
	case "sint32":
		return mapZigzagAssign(target, "ProtoCoreUtils.decode_zigzag32")
	case "sint64":
		return mapZigzagAssign(target, "ProtoCoreUtils.decode_zigzag64")
	case "float":
		return mapFixedAssign(target, "ProtoCoreUtils.decode_float", 4)
	case "double":
		return mapFixedAssign(target, "ProtoCoreUtils.decode_double", 8)
	case "fixed32":
		return mapFixedAssign(target, "ProtoCoreUtils.decode_fixed32", 4)
	case "sfixed32":
		return mapFixedAssign(target, "ProtoCoreUtils.decode_sfixed32", 4)
	case "fixed64":
		return mapFixedAssign(target, "ProtoCoreUtils.decode_fixed64", 8)
	case "sfixed64":
		return mapFixedAssign(target, "ProtoCoreUtils.decode_sfixed64", 8)
	case "string":
		return mapStringAssign(target)
	case "bytes":
		return mapBytesAssign(target)
	default:
		// Map values may be messages.
		return mapMessageAssign(target, protoType)
	}
}

// mapVarintAssign decodes a varint from the entry buffer into target.
func mapVarintAssign(target string, asBool bool) []gdast.Statement {
	stmts := []gdast.Statement{
		inferredVar("result", "ProtoCoreUtils.decode_varint(entry_data, entry_offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
	}
	if asBool {
		stmts = append(stmts, rawf("%s = result.value != 0", target))
	} else {
		stmts = append(stmts, rawf("%s = result.value", target))
	}
	stmts = append(stmts, rawf("entry_offset += result.size"))
	return stmts
}

// mapZigzagAssign decodes a zig-zag varint from the entry buffer into target.
func mapZigzagAssign(target, decodeFunc string) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("result", "ProtoCoreUtils.decode_varint(entry_data, entry_offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.VARINT_NOT_FOUND"}),
			},
		},
		rawf("%s = %s(result.value)", target, decodeFunc),
		rawf("entry_offset += result.size"),
	}
}

// mapFixedAssign decodes a fixed-width numeric from the entry buffer into
// target.
func mapFixedAssign(target, decodeFunc string, width int) []gdast.Statement {
	return []gdast.Statement{
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: fmt.Sprintf("entry_offset + %d > entry_data.size()", width)},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.PARSE_INCOMPLETE"}),
			},
		},
		rawf("%s = %s(entry_data, entry_offset)", target, decodeFunc),
		rawf("entry_offset += %d", width),
	}
}

// mapStringAssign decodes a length-prefixed string from the entry buffer into
// target.
func mapStringAssign(target string) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("len_result", "ProtoCoreUtils.decode_varint(entry_data, entry_offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "len_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND"}),
			},
		},
		rawf("entry_offset += len_result.size"),
		gdast.VarDeclaration{
			Name:         "str_len",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "len_result.value"},
		},
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: "entry_offset + str_len > entry_data.size()"},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"}),
			},
		},
		rawf("%s = ProtoCoreUtils.decode_string(entry_data, entry_offset, str_len)", target),
		rawf("entry_offset += str_len"),
	}
}

// mapBytesAssign decodes a length-prefixed byte slice from the entry buffer
// into target.
func mapBytesAssign(target string) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("len_result", "ProtoCoreUtils.decode_varint(entry_data, entry_offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "len_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND"}),
			},
		},
		rawf("entry_offset += len_result.size"),
		gdast.VarDeclaration{
			Name:         "byte_len",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "len_result.value"},
		},
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: "entry_offset + byte_len > entry_data.size()"},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"}),
			},
		},
		rawf("%s = entry_data.slice(entry_offset, entry_offset + byte_len)", target),
		rawf("entry_offset += byte_len"),
	}
}

// mapMessageAssign decodes a length-prefixed embedded message from the entry
// buffer into target by constructing a new instance of the value's type and
// calling from_bytes; nested errors are propagated.
func mapMessageAssign(target, typeName string) []gdast.Statement {
	return []gdast.Statement{
		inferredVar("len_result", "ProtoCoreUtils.decode_varint(entry_data, entry_offset)"),
		gdast.IfStatement{
			Condition: gdast.Eq(gdast.RawExpression{Code: "len_result.size"}, gdast.Lit(-1)),
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_NOT_FOUND"}),
			},
		},
		rawf("entry_offset += len_result.size"),
		gdast.VarDeclaration{
			Name:         "msg_len",
			TypeHint:     "int",
			InitialValue: gdast.RawExpression{Code: "len_result.value"},
		},
		gdast.IfStatement{
			Condition: gdast.RawExpression{Code: "entry_offset + msg_len > entry_data.size()"},
			Body: []gdast.Statement{
				gdast.Ret(gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.LENGTH_DELIMITED_SIZE_MISMATCH"}),
			},
		},
		gdast.VarDeclaration{
			Name:         "entry_msg_data",
			TypeHint:     "PackedByteArray",
			InitialValue: gdast.RawExpression{Code: "entry_data.slice(entry_offset, entry_offset + msg_len)"},
		},
		rawf("%s = %s.new()", target, typeName),
		inferredVar("entry_msg_result", fmt.Sprintf("%s.from_bytes(entry_msg_data)", target)),
		gdast.IfStatement{
			Condition: gdast.Ne(gdast.V("entry_msg_result"), gdast.RawExpression{Code: "ProtoCoreUtils.ProtobufError.NO_ERRORS"}),
			Body: []gdast.Statement{
				gdast.Ret(gdast.V("entry_msg_result")),
			},
		},
		rawf("entry_offset += msg_len"),
	}
}

// mapDefault returns the literal zero value used to initialize the running
// `map_key` / `map_value` variables before the entry-loop populates them.
func mapDefault(protoType string) string {
	if def, ok := scalarDefaultMap[protoType]; ok {
		return def
	}
	return "null"
}
