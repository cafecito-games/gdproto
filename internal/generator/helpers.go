package generator

// Wire type values from the protobuf wire format specification.
const (
	wireTypeVarint          = 0
	wireTypeFixed64         = 1
	wireTypeLengthDelimited = 2
	wireTypeFixed32         = 5
)

// wireTypeMap maps proto3 scalar type names to their wire type codes. Types
// not present in this map (messages, enum-typed fields stored as references)
// fall back to the length-delimited wire type via wireType.
var wireTypeMap = map[string]int{
	"double":   wireTypeFixed64,
	"float":    wireTypeFixed32,
	"int32":    wireTypeVarint,
	"int64":    wireTypeVarint,
	"uint32":   wireTypeVarint,
	"uint64":   wireTypeVarint,
	"sint32":   wireTypeVarint,
	"sint64":   wireTypeVarint,
	"fixed32":  wireTypeFixed32,
	"fixed64":  wireTypeFixed64,
	"sfixed32": wireTypeFixed32,
	"sfixed64": wireTypeFixed64,
	"bool":     wireTypeVarint,
	"string":   wireTypeLengthDelimited,
	"bytes":    wireTypeLengthDelimited,
}

// wireType returns the wire-type code for the given proto type name. Unknown
// types (typically message references) use the length-delimited wire type.
func wireType(protoType string) int {
	if t, ok := wireTypeMap[protoType]; ok {
		return t
	}
	return wireTypeLengthDelimited
}
