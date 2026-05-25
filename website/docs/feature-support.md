---
title: Feature support
description: Supported and unsupported protobuf features.
---

# Feature support

gdproto targets proto3 schemas and generates GDScript wrappers for data
messages. It does not generate service clients or servers.

## Supported

| Feature | Notes |
| --- | --- |
| Scalar fields | `int32`, `int64`, `uint32`, `uint64`, `sint32`, `sint64`, `fixed32`, `fixed64`, `sfixed32`, `sfixed64`, `float`, `double`, `bool`, `string`, and `bytes`. |
| Messages | Top-level and nested messages are generated as GDScript classes. |
| Enums | Top-level and nested enums are generated as GDScript enums. Alias, negative, decimal, hex, and octal enum values are parsed. |
| Repeated fields | Scalar and message repeated fields are supported. Primitive repeated fields use packed encoding by default where proto3 does. |
| Maps | Scalar keys and any supported value type are supported. |
| Oneofs | Oneof setters update a generated discriminant enum and clear the previous member. |
| Imports | Package-qualified, absolute, transitive public imports, and cycles with detection are handled by the importer/plugin path. |
| Reserved values | Reserved numbers, ranges, and names are validated. |
| Options | File, message, and field options are parsed. Field-level `packed = false` is honored. |
| Proto3 optional | Explicit presence with the `optional` keyword is supported. |
| Binary wire format | Generated binary output is compatible with standard protobuf decoders for supported features. |
| Text format | Generated wrappers include gdproto text-format serialization and parsing for round trips and debugging. |

## Supported With Caveats

| Feature | Caveat |
| --- | --- |
| Well-known types | They can be treated as ordinary imported message schemas when their `.proto` descriptors are available and generated. gdproto does not provide special Godot-native mappings for types such as `Timestamp` or `Duration`. |
| Imported message references | Generated GDScript references imported messages through their generated wrapper classes, so imported proto wrappers must exist in the Godot project. |
| JSON names and JSON mapping | JSON names may be parsed as options, but gdproto does not generate protobuf JSON mapping helpers. |

## Not Supported

| Feature | Behavior |
| --- | --- |
| proto2 syntax | Rejected by validation. |
| Services and RPCs | Not generated. |
| Custom options | Parsed as options where possible, but custom option semantics are not implemented. The one exception is the `(gdproto.class_prefix)` file option, which the generator reads to override the per-file class prefix — see [Generated GDScript](./generated-code.md#class-prefix). |
| Extensions | Not implemented. |
| Protobuf JSON mapping | Not generated. Use binary wire format or gdproto text format. |
