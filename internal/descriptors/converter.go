package descriptors

import (
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/cafecito-games/gogdproto/internal/ast"
)

// scalarTypeNames maps the wire-type enum to the proto scalar type spelling.
// TYPE_MESSAGE, TYPE_ENUM, and TYPE_GROUP are not in this map; they use TypeName.
var scalarTypeNames = map[descriptorpb.FieldDescriptorProto_Type]string{
	descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:   "double",
	descriptorpb.FieldDescriptorProto_TYPE_FLOAT:    "float",
	descriptorpb.FieldDescriptorProto_TYPE_INT64:    "int64",
	descriptorpb.FieldDescriptorProto_TYPE_UINT64:   "uint64",
	descriptorpb.FieldDescriptorProto_TYPE_INT32:    "int32",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED64:  "fixed64",
	descriptorpb.FieldDescriptorProto_TYPE_FIXED32:  "fixed32",
	descriptorpb.FieldDescriptorProto_TYPE_BOOL:     "bool",
	descriptorpb.FieldDescriptorProto_TYPE_STRING:   "string",
	descriptorpb.FieldDescriptorProto_TYPE_BYTES:    "bytes",
	descriptorpb.FieldDescriptorProto_TYPE_UINT32:   "uint32",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED32: "sfixed32",
	descriptorpb.FieldDescriptorProto_TYPE_SFIXED64: "sfixed64",
	descriptorpb.FieldDescriptorProto_TYPE_SINT32:   "sint32",
	descriptorpb.FieldDescriptorProto_TYPE_SINT64:   "sint64",
}

// converter holds cross-file state used while converting descriptors.
type converter struct {
	// typeRegistry maps a fully-qualified type path (no leading dot) to the
	// proto file in which it was declared. Used to populate Field.SourceFile.
	typeRegistry map[string]string
}

// FromCodeGeneratorRequest converts every FileDescriptorProto in the request
// (proto_file) to a ProtoFile AST. The full proto_file list is used to build
// a cross-file type registry so that message/enum field references resolve to
// their source file path.
func FromCodeGeneratorRequest(req *pluginpb.CodeGeneratorRequest) ([]*ast.ProtoFile, error) {
	c := newConverter(req.GetProtoFile())
	out := make([]*ast.ProtoFile, 0, len(req.GetProtoFile()))
	for _, fdp := range req.GetProtoFile() {
		file, err := c.convertFile(fdp)
		if err != nil {
			return nil, err
		}
		out = append(out, file)
	}
	return out, nil
}

// FromFileDescriptorProto converts a single FileDescriptorProto to a
// ProtoFile AST. Cross-file source-file resolution is limited to types defined
// within the given descriptor.
func FromFileDescriptorProto(fdp *descriptorpb.FileDescriptorProto) (*ast.ProtoFile, error) {
	c := newConverter([]*descriptorpb.FileDescriptorProto{fdp})
	return c.convertFile(fdp)
}

func newConverter(allFiles []*descriptorpb.FileDescriptorProto) *converter {
	c := &converter{typeRegistry: map[string]string{}}
	for _, fd := range allFiles {
		pkg := fd.GetPackage()
		source := fd.GetName()
		for _, m := range fd.GetMessageType() {
			c.registerMessage(m, pkg, source, "")
		}
		for _, e := range fd.GetEnumType() {
			full := e.GetName()
			if pkg != "" {
				full = pkg + "." + full
			}
			c.typeRegistry[full] = source
		}
	}
	return c
}

func (c *converter) registerMessage(m *descriptorpb.DescriptorProto, pkg, source, parent string) {
	var full string
	switch {
	case parent != "":
		full = parent + "." + m.GetName()
	case pkg != "":
		full = pkg + "." + m.GetName()
	default:
		full = m.GetName()
	}
	c.typeRegistry[full] = source
	for _, nested := range m.GetNestedType() {
		if nested.GetOptions().GetMapEntry() {
			continue
		}
		c.registerMessage(nested, pkg, source, full)
	}
	for _, e := range m.GetEnumType() {
		c.typeRegistry[full+"."+e.GetName()] = source
	}
}

func (c *converter) convertFile(fd *descriptorpb.FileDescriptorProto) (*ast.ProtoFile, error) {
	syntax := fd.GetSyntax()
	if syntax == "" {
		syntax = "proto3"
	}
	file := &ast.ProtoFile{
		Syntax:  syntax,
		Package: fd.GetPackage(),
		Options: map[string]any{},
	}

	publicSet := map[int32]struct{}{}
	for _, idx := range fd.GetPublicDependency() {
		publicSet[idx] = struct{}{}
	}
	for i, dep := range fd.GetDependency() {
		_, public := publicSet[int32(i)]
		file.Imports = append(file.Imports, &ast.Import{Path: dep, Public: public})
	}

	for _, e := range fd.GetEnumType() {
		file.Enums = append(file.Enums, c.convertEnum(e))
	}
	for _, m := range fd.GetMessageType() {
		msg, err := c.convertMessage(m)
		if err != nil {
			return nil, err
		}
		file.Messages = append(file.Messages, msg)
	}
	return file, nil
}

func (c *converter) convertMessage(d *descriptorpb.DescriptorProto) (*ast.Message, error) {
	msg := &ast.Message{
		Name:    d.GetName(),
		Options: map[string]any{},
	}

	// Index nested map-entry types so we can dispatch map fields.
	mapEntries := map[string]*descriptorpb.DescriptorProto{}
	for _, nested := range d.GetNestedType() {
		if nested.GetOptions().GetMapEntry() {
			mapEntries[nested.GetName()] = nested
		}
	}

	oneofNames := make([]string, len(d.GetOneofDecl()))
	for i, o := range d.GetOneofDecl() {
		oneofNames[i] = o.GetName()
	}

	var regularFields []*ast.Field
	var mapFields []*ast.MapField

	for _, f := range d.GetField() {
		if f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED &&
			f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			short := lastSegment(f.GetTypeName())
			if entry, ok := mapEntries[short]; ok {
				mf, err := c.convertMapField(f, entry)
				if err != nil {
					return nil, err
				}
				mapFields = append(mapFields, mf)
				continue
			}
		}

		field := c.convertField(f)
		if f.OneofIndex != nil {
			idx := int(f.GetOneofIndex())
			if idx >= 0 && idx < len(oneofNames) {
				field.OneofParent = oneofNames[idx]
			}
		}
		regularFields = append(regularFields, field)
	}

	// Convert oneofs. Synthetic oneofs (proto3 optional) are not real oneofs:
	// Python suppresses them and keeps the field as a regular optional field.
	var oneofs []*ast.Oneof
	oneofFieldSet := map[*ast.Field]struct{}{}
	for _, o := range d.GetOneofDecl() {
		var fields []*ast.Field
		for _, f := range regularFields {
			if f.OneofParent == o.GetName() {
				fields = append(fields, f)
			}
		}
		isSynthetic := len(fields) == 1 && fields[0].Optional && strings.HasPrefix(o.GetName(), "_")
		if isSynthetic {
			fields[0].OneofParent = ""
			continue
		}
		for _, f := range fields {
			oneofFieldSet[f] = struct{}{}
		}
		oneofs = append(oneofs, &ast.Oneof{
			Name:    o.GetName(),
			Fields:  fields,
			Options: map[string]any{},
		})
	}

	// Strip oneof-owned fields from the regular field list.
	if len(oneofFieldSet) > 0 {
		filtered := regularFields[:0]
		for _, f := range regularFields {
			if _, isOneof := oneofFieldSet[f]; !isOneof {
				filtered = append(filtered, f)
			}
		}
		regularFields = filtered
	}

	msg.Fields = regularFields
	msg.Maps = mapFields
	msg.Oneofs = oneofs

	for _, nested := range d.GetNestedType() {
		if nested.GetOptions().GetMapEntry() {
			continue
		}
		nm, err := c.convertMessage(nested)
		if err != nil {
			return nil, err
		}
		msg.NestedMessages = append(msg.NestedMessages, nm)
	}
	for _, e := range d.GetEnumType() {
		msg.NestedEnums = append(msg.NestedEnums, c.convertEnum(e))
	}

	// Reserved ranges: descriptor end is exclusive; AST uses inclusive.
	var ranges []ast.ReservedRange
	for _, r := range d.GetReservedRange() {
		start := int(r.GetStart())
		end := int(r.GetEnd()) - 1
		ranges = append(ranges, ast.ReservedRange{Start: start, End: end})
	}
	if len(ranges) > 0 || len(d.GetReservedName()) > 0 {
		msg.Reserved = []*ast.Reserved{{
			Numbers: ranges,
			Names:   append([]string(nil), d.GetReservedName()...),
		}}
	}

	return msg, nil
}

func (c *converter) convertField(f *descriptorpb.FieldDescriptorProto) *ast.Field {
	field := &ast.Field{
		Name:     f.GetName(),
		Number:   int(f.GetNumber()),
		Repeated: f.GetLabel() == descriptorpb.FieldDescriptorProto_LABEL_REPEATED,
		Optional: f.GetProto3Optional(),
		Options:  map[string]any{},
	}

	switch f.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		fullPath := strings.TrimPrefix(f.GetTypeName(), ".")
		field.FullTypePath = fullPath
		field.FieldType = lastSegment(fullPath)
		field.SourceFile = c.typeRegistry[fullPath]
		field.IsEnum = f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
	default:
		if name, ok := scalarTypeNames[f.GetType()]; ok {
			field.FieldType = name
		} else {
			field.FieldType = "unknown"
		}
	}

	if opts := f.GetOptions(); opts != nil && opts.Packed != nil {
		field.Options["packed"] = opts.GetPacked()
	}

	return field
}

func (c *converter) convertEnum(e *descriptorpb.EnumDescriptorProto) *ast.Enum {
	out := &ast.Enum{
		Name:    e.GetName(),
		Options: map[string]any{},
	}
	for _, v := range e.GetValue() {
		out.Values = append(out.Values, &ast.EnumValue{
			Name:    v.GetName(),
			Number:  int(v.GetNumber()),
			Options: map[string]any{},
		})
	}
	if e.GetOptions().GetAllowAlias() {
		out.Options["allow_alias"] = true
	}
	return out
}

func (c *converter) convertMapField(f *descriptorpb.FieldDescriptorProto, entry *descriptorpb.DescriptorProto) (*ast.MapField, error) {
	if len(entry.GetField()) != 2 {
		return nil, &mapEntryError{name: f.GetName()}
	}
	var keyDescriptor, valueDescriptor *descriptorpb.FieldDescriptorProto
	for _, ef := range entry.GetField() {
		switch ef.GetName() {
		case "key":
			keyDescriptor = ef
		case "value":
			valueDescriptor = ef
		}
	}
	if keyDescriptor == nil || valueDescriptor == nil {
		// Fall back to positional: descriptors guarantee key=1, value=2.
		keyDescriptor = entry.GetField()[0]
		valueDescriptor = entry.GetField()[1]
	}

	mf := &ast.MapField{
		Name:    f.GetName(),
		Number:  int(f.GetNumber()),
		Options: map[string]any{},
	}

	if name, ok := scalarTypeNames[keyDescriptor.GetType()]; ok {
		mf.KeyType = name
	} else {
		mf.KeyType = "unknown"
	}

	switch valueDescriptor.GetType() {
	case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		fullPath := strings.TrimPrefix(valueDescriptor.GetTypeName(), ".")
		mf.FullValueTypePath = fullPath
		mf.ValueType = lastSegment(fullPath)
		mf.ValueSourceFile = c.typeRegistry[fullPath]
		mf.ValueIsEnum = valueDescriptor.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM
	default:
		if name, ok := scalarTypeNames[valueDescriptor.GetType()]; ok {
			mf.ValueType = name
		} else {
			mf.ValueType = "unknown"
		}
	}

	return mf, nil
}

type mapEntryError struct {
	name string
}

func (e *mapEntryError) Error() string {
	return "invalid map entry for field " + e.name
}

// lastSegment returns the substring after the last '.' in name, or the whole
// string if there is no '.'. Used to pull a short type name out of a
// descriptor's fully-qualified TypeName (e.g., ".pkg.Outer.Entry" -> "Entry").
func lastSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}
