package descriptors

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

func strPtr(s string) *string { return &s }
func i32Ptr(v int32) *int32   { return &v }
func boolPtr(b bool) *bool    { return &b }

func labelPtr(l descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto_Label {
	return &l
}
func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}

func TestConvertEmptyFile(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("empty.proto"),
		Package: strPtr("demo"),
		Syntax:  strPtr("proto3"),
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Syntax != "proto3" {
		t.Errorf("Syntax = %q, want proto3", got.Syntax)
	}
	if got.Package != "demo" {
		t.Errorf("Package = %q, want demo", got.Package)
	}
	if len(got.Messages) != 0 || len(got.Enums) != 0 || len(got.Imports) != 0 {
		t.Errorf("expected empty file; got %+v", got)
	}
}

func TestConvertEmptyFileDefaultsToProto3(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{Name: strPtr("a.proto")}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Syntax != "proto3" {
		t.Errorf("default syntax = %q, want proto3", got.Syntax)
	}
}

func TestConvertImports(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:             strPtr("a.proto"),
		Dependency:       []string{"common.proto", "x/y.proto"},
		PublicDependency: []int32{1},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got.Imports) != 2 {
		t.Fatalf("imports len = %d, want 2", len(got.Imports))
	}
	if got.Imports[0].Path != "common.proto" || got.Imports[0].Public {
		t.Errorf("import 0 = %+v", got.Imports[0])
	}
	if got.Imports[1].Path != "x/y.proto" || !got.Imports[1].Public {
		t.Errorf("import 1 = %+v", got.Imports[1])
	}
}

func TestConvertScalarFields(t *testing.T) {
	cases := []struct {
		name     string
		t        descriptorpb.FieldDescriptorProto_Type
		expected string
	}{
		{"d", descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, "double"},
		{"f", descriptorpb.FieldDescriptorProto_TYPE_FLOAT, "float"},
		{"i64", descriptorpb.FieldDescriptorProto_TYPE_INT64, "int64"},
		{"u64", descriptorpb.FieldDescriptorProto_TYPE_UINT64, "uint64"},
		{"i32", descriptorpb.FieldDescriptorProto_TYPE_INT32, "int32"},
		{"fx64", descriptorpb.FieldDescriptorProto_TYPE_FIXED64, "fixed64"},
		{"fx32", descriptorpb.FieldDescriptorProto_TYPE_FIXED32, "fixed32"},
		{"b", descriptorpb.FieldDescriptorProto_TYPE_BOOL, "bool"},
		{"s", descriptorpb.FieldDescriptorProto_TYPE_STRING, "string"},
		{"by", descriptorpb.FieldDescriptorProto_TYPE_BYTES, "bytes"},
		{"u32", descriptorpb.FieldDescriptorProto_TYPE_UINT32, "uint32"},
		{"sf32", descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, "sfixed32"},
		{"sf64", descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, "sfixed64"},
		{"si32", descriptorpb.FieldDescriptorProto_TYPE_SINT32, "sint32"},
		{"si64", descriptorpb.FieldDescriptorProto_TYPE_SINT64, "sint64"},
	}

	var fields []*descriptorpb.FieldDescriptorProto
	for i, c := range cases {
		fields = append(fields, &descriptorpb.FieldDescriptorProto{
			Name:   strPtr(c.name),
			Number: i32Ptr(int32(i + 1)),
			Type:   typePtr(c.t),
			Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
		})
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("scalars.proto"),
		Syntax: strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("M"), Field: fields},
		},
	}

	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msg := got.Messages[0]
	if len(msg.Fields) != len(cases) {
		t.Fatalf("field count = %d, want %d", len(msg.Fields), len(cases))
	}
	for i, c := range cases {
		f := msg.Fields[i]
		if f.Name != c.name || f.FieldType != c.expected {
			t.Errorf("field %d: got name=%q type=%q, want %q/%q", i, f.Name, f.FieldType, c.name, c.expected)
		}
		if f.Number != i+1 {
			t.Errorf("field %d: number = %d, want %d", i, f.Number, i+1)
		}
		if f.Repeated {
			t.Errorf("field %d: unexpectedly Repeated", i)
		}
	}
}

func TestConvertMessageField(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("msg.proto"),
		Package: strPtr("demo"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Inner")},
			{
				Name: strPtr("Outer"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("inner"),
						Number:   i32Ptr(1),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".demo.Inner"),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	outer := got.Messages[1]
	f := outer.Fields[0]
	if f.FieldType != "Inner" {
		t.Errorf("FieldType = %q, want Inner", f.FieldType)
	}
	if f.FullTypePath != "demo.Inner" {
		t.Errorf("FullTypePath = %q, want demo.Inner", f.FullTypePath)
	}
	if f.SourceFile != "msg.proto" {
		t.Errorf("SourceFile = %q, want msg.proto", f.SourceFile)
	}
	if f.IsEnum {
		t.Errorf("IsEnum should be false for message field")
	}
}

func TestConvertEnumField(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("e.proto"),
		Package: strPtr("demo"),
		Syntax:  strPtr("proto3"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Color"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("RED"), Number: i32Ptr(0)},
					{Name: strPtr("GREEN"), Number: i32Ptr(1)},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Paint"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("color"),
						Number:   i32Ptr(1),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
						TypeName: strPtr(".demo.Color"),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Enums[0].Name != "Color" || len(got.Enums[0].Values) != 2 {
		t.Errorf("enum mismatch: %+v", got.Enums[0])
	}
	f := got.Messages[0].Fields[0]
	if !f.IsEnum {
		t.Errorf("expected enum field")
	}
	if f.FieldType != "Color" {
		t.Errorf("FieldType = %q", f.FieldType)
	}
}

func TestConvertRepeatedField(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("r.proto"),
		Syntax: strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("M"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   strPtr("tags"),
						Number: i32Ptr(1),
						Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	f := got.Messages[0].Fields[0]
	if !f.Repeated {
		t.Errorf("expected Repeated=true")
	}
	if f.FieldType != "string" {
		t.Errorf("FieldType = %q", f.FieldType)
	}
}

func TestConvertEnum(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("e.proto"),
		Syntax: strPtr("proto3"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: strPtr("Status"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: strPtr("UNSET"), Number: i32Ptr(0)},
					{Name: strPtr("OK"), Number: i32Ptr(1)},
				},
				Options: &descriptorpb.EnumOptions{AllowAlias: boolPtr(true)},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	e := got.Enums[0]
	if e.Name != "Status" || len(e.Values) != 2 {
		t.Errorf("enum: %+v", e)
	}
	if e.Values[1].Name != "OK" || e.Values[1].Number != 1 {
		t.Errorf("enum values mismatch: %+v", e.Values)
	}
	if alias, _ := e.Options["allow_alias"].(bool); !alias {
		t.Errorf("allow_alias not propagated")
	}
}

func TestConvertNestedMessageAndEnum(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("n.proto"),
		Syntax: strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("Outer"),
				NestedType: []*descriptorpb.DescriptorProto{
					{Name: strPtr("Inner")},
				},
				EnumType: []*descriptorpb.EnumDescriptorProto{
					{
						Name: strPtr("Kind"),
						Value: []*descriptorpb.EnumValueDescriptorProto{
							{Name: strPtr("A"), Number: i32Ptr(0)},
						},
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	outer := got.Messages[0]
	if len(outer.NestedMessages) != 1 || outer.NestedMessages[0].Name != "Inner" {
		t.Errorf("nested messages: %+v", outer.NestedMessages)
	}
	if len(outer.NestedEnums) != 1 || outer.NestedEnums[0].Name != "Kind" {
		t.Errorf("nested enums: %+v", outer.NestedEnums)
	}
}

func TestConvertOneof(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("o.proto"),
		Syntax: strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("M"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: strPtr("choice")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:       strPtr("a"),
						Number:     i32Ptr(1),
						Type:       typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						Label:      labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						OneofIndex: i32Ptr(0),
					},
					{
						Name:       strPtr("b"),
						Number:     i32Ptr(2),
						Type:       typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT32),
						Label:      labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						OneofIndex: i32Ptr(0),
					},
					{
						Name:   strPtr("name"),
						Number: i32Ptr(3),
						Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msg := got.Messages[0]
	if len(msg.Fields) != 1 || msg.Fields[0].Name != "name" {
		t.Errorf("regular fields: %+v", msg.Fields)
	}
	if len(msg.Oneofs) != 1 {
		t.Fatalf("oneofs len = %d, want 1", len(msg.Oneofs))
	}
	o := msg.Oneofs[0]
	if o.Name != "choice" || len(o.Fields) != 2 {
		t.Errorf("oneof: %+v", o)
	}
	if o.Fields[0].Name != "a" || o.Fields[1].Name != "b" {
		t.Errorf("oneof fields: %+v", o.Fields)
	}
	if o.Fields[0].OneofParent != "choice" {
		t.Errorf("OneofParent = %q", o.Fields[0].OneofParent)
	}
}

func TestConvertProto3OptionalIsNotOneof(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("po.proto"),
		Syntax: strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("M"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: strPtr("_nickname")},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:           strPtr("nickname"),
						Number:         i32Ptr(1),
						Type:           typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						Label:          labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						OneofIndex:     i32Ptr(0),
						Proto3Optional: boolPtr(true),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msg := got.Messages[0]
	if len(msg.Oneofs) != 0 {
		t.Errorf("synthetic oneof leaked: %+v", msg.Oneofs)
	}
	if len(msg.Fields) != 1 {
		t.Fatalf("fields = %d", len(msg.Fields))
	}
	if !msg.Fields[0].Optional || msg.Fields[0].OneofParent != "" {
		t.Errorf("expected optional regular field, got %+v", msg.Fields[0])
	}
}

func TestConvertMapField(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("m.proto"),
		Package: strPtr("demo"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("M"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name:    strPtr("AttrsEntry"),
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   strPtr("key"),
								Number: i32Ptr(1),
								Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
								Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
							},
							{
								Name:   strPtr("value"),
								Number: i32Ptr(2),
								Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT32),
								Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
							},
						},
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("attrs"),
						Number:   i32Ptr(1),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".demo.M.AttrsEntry"),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	msg := got.Messages[0]
	if len(msg.Fields) != 0 {
		t.Errorf("unexpected regular fields: %+v", msg.Fields)
	}
	if len(msg.NestedMessages) != 0 {
		t.Errorf("map-entry nested type leaked into nested messages")
	}
	if len(msg.Maps) != 1 {
		t.Fatalf("maps len = %d, want 1", len(msg.Maps))
	}
	mf := msg.Maps[0]
	if mf.Name != "attrs" || mf.KeyType != "string" || mf.ValueType != "int32" {
		t.Errorf("map field mismatch: %+v", mf)
	}
}

func TestConvertMapFieldWithMessageValue(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("mv.proto"),
		Package: strPtr("demo"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("Item")},
			{
				Name: strPtr("Bag"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name:    strPtr("ItemsEntry"),
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:   strPtr("key"),
								Number: i32Ptr(1),
								Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
								Label:  labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
							},
							{
								Name:     strPtr("value"),
								Number:   i32Ptr(2),
								Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
								TypeName: strPtr(".demo.Item"),
								Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
							},
						},
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     strPtr("items"),
						Number:   i32Ptr(1),
						Type:     typePtr(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
						TypeName: strPtr(".demo.Bag.ItemsEntry"),
						Label:    labelPtr(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
					},
				},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	bag := got.Messages[1]
	mf := bag.Maps[0]
	if mf.ValueType != "Item" || mf.FullValueTypePath != "demo.Item" {
		t.Errorf("map value type: %+v", mf)
	}
	if mf.ValueSourceFile != "mv.proto" {
		t.Errorf("ValueSourceFile = %q", mf.ValueSourceFile)
	}
	if mf.ValueIsEnum {
		t.Errorf("ValueIsEnum should be false")
	}
}

func TestConvertReserved(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:   strPtr("r.proto"),
		Syntax: strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("M"),
				ReservedRange: []*descriptorpb.DescriptorProto_ReservedRange{
					{Start: i32Ptr(5), End: i32Ptr(6)},   // single number 5
					{Start: i32Ptr(10), End: i32Ptr(15)}, // 10..14
				},
				ReservedName: []string{"foo", "bar"},
			},
		},
	}
	got, err := FromFileDescriptorProto(fdp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	res := got.Messages[0].Reserved
	if len(res) != 1 {
		t.Fatalf("reserved len = %d", len(res))
	}
	if len(res[0].Numbers) != 2 {
		t.Fatalf("numbers len = %d", len(res[0].Numbers))
	}
	if res[0].Numbers[0].Start != 5 || res[0].Numbers[0].End != 5 {
		t.Errorf("range 0: %+v", res[0].Numbers[0])
	}
	if res[0].Numbers[1].Start != 10 || res[0].Numbers[1].End != 14 {
		t.Errorf("range 1: %+v", res[0].Numbers[1])
	}
	if len(res[0].Names) != 2 || res[0].Names[0] != "foo" {
		t.Errorf("names: %+v", res[0].Names)
	}
}

func TestFromCodeGeneratorRequest(t *testing.T) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"a.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			{
				Name:    proto.String("a.proto"),
				Syntax:  proto.String("proto3"),
				Package: proto.String("p"),
				MessageType: []*descriptorpb.DescriptorProto{
					{Name: proto.String("X")},
				},
			},
		},
	}
	files, err := FromCodeGeneratorRequest(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(files) != 1 || files[0].Package != "p" || files[0].Messages[0].Name != "X" {
		t.Errorf("unexpected: %+v", files)
	}
}
