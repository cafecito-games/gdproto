package validator

// Field-number bounds defined by the protobuf specification.
//
//nolint:unused // consumed by subsequent validator tasks.
const (
	minFieldNumber = 1
	maxFieldNumber = 536870911 // 2^29 - 1
	reservedStart  = 19000
	reservedEnd    = 19999
)

// validMapKeyTypes lists the integral and string types allowed as map keys.
//
//nolint:unused // consumed by subsequent validator tasks.
var validMapKeyTypes = map[string]bool{
	"int32":    true,
	"int64":    true,
	"uint32":   true,
	"uint64":   true,
	"sint32":   true,
	"sint64":   true,
	"fixed32":  true,
	"fixed64":  true,
	"sfixed32": true,
	"sfixed64": true,
	"bool":     true,
	"string":   true,
}

// scalarTypes lists every scalar type recognised by proto3.
//
//nolint:unused // consumed by subsequent validator tasks.
var scalarTypes = map[string]bool{
	"double":   true,
	"float":    true,
	"int32":    true,
	"int64":    true,
	"uint32":   true,
	"uint64":   true,
	"sint32":   true,
	"sint64":   true,
	"fixed32":  true,
	"fixed64":  true,
	"sfixed32": true,
	"sfixed64": true,
	"bool":     true,
	"string":   true,
	"bytes":    true,
}

// reservedKeywords lists protobuf keywords that cannot be used as identifiers.
// Comparisons against this set are case-insensitive (lookup with lowercased key).
var reservedKeywords = map[string]bool{
	"syntax":   true,
	"message":  true,
	"enum":     true,
	"repeated": true,
	"map":      true,
	"oneof":    true,
	"import":   true,
	"public":   true,
	"option":   true,
	"packed":   true,
	"reserved": true,
	"package":  true,
	"service":  true,
	"rpc":      true,
	"returns":  true,
	"stream":   true,
	"double":   true,
	"float":    true,
	"int32":    true,
	"int64":    true,
	"uint32":   true,
	"uint64":   true,
	"sint32":   true,
	"sint64":   true,
	"fixed32":  true,
	"fixed64":  true,
	"sfixed32": true,
	"sfixed64": true,
	"bool":     true,
	"string":   true,
	"bytes":    true,
	"true":     true,
	"false":    true,
}
