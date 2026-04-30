package ast

// Position is the source-code location of an AST node (1-indexed).
type Position struct {
	Line   int
	Column int
}

// ProtoFile represents a complete .proto file.
type ProtoFile struct {
	Position
	Syntax   string
	Imports  []*Import
	Package  string
	Messages []*Message
	Enums    []*Enum
	Options  map[string]any
}

// Import represents an `import "..."` statement.
type Import struct {
	Position
	Path   string
	Public bool
}

// Message represents a `message Name { ... }` definition.
type Message struct {
	Position
	Name           string
	Fields         []*Field
	NestedMessages []*Message
	NestedEnums    []*Enum
	Oneofs         []*Oneof
	Maps           []*MapField
	Reserved       []*Reserved
	Options        map[string]any
}

// Field represents a regular field in a message.
type Field struct {
	Position
	FieldType    string
	Name         string
	Number       int
	Repeated     bool
	Optional     bool
	Options      map[string]any
	OneofParent  string // empty when not in a oneof
	FullTypePath string
	SourceFile   string
	IsEnum       bool
	// EnumValues mirrors the resolved enum's values when this field references
	// an enum defined in another file. The local-file generator already has
	// access to the AST node, so this is left empty for same-file enums.
	EnumValues []*EnumValue
}

// MapField represents a map<K,V> field.
type MapField struct {
	Position
	KeyType           string
	ValueType         string
	Name              string
	Number            int
	Options           map[string]any
	FullValueTypePath string
	ValueSourceFile   string
	ValueIsEnum       bool
}

// Oneof represents a oneof group.
type Oneof struct {
	Position
	Name    string
	Fields  []*Field
	Options map[string]any
}

// Enum represents an enum definition.
type Enum struct {
	Position
	Name    string
	Values  []*EnumValue
	Options map[string]any
}

// EnumValue is a single `NAME = N;` line inside an enum.
type EnumValue struct {
	Position
	Name    string
	Number  int
	Options map[string]any
}

// Reserved represents a `reserved 1, 2 to 5;` or `reserved "foo";` line.
// Numbers carries integer reservations (single numbers as ranges with
// Start == End); Names carries string-name reservations.
type Reserved struct {
	Position
	Numbers []ReservedRange
	Names   []string
}

// ReservedRange is a [Start, End] inclusive range. For a single reserved
// number, Start == End.
type ReservedRange struct {
	Start int
	End   int
}

// Option represents an `option name = value;` statement. Value carries
// the parsed value: string, int64, float64, bool, or identifier (string).
type Option struct {
	Position
	Name  string
	Value any
}
