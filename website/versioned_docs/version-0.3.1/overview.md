---
title: Overview
description: What gdproto generates and how the pieces fit together.
---

# gdproto

`gdproto` compiles Protocol Buffers v3 schemas into GDScript for Godot 4.6+.
It is distributed as Go binaries and has no GDScript runtime dependency beyond
the generated `proto_core_utils.gd` file.

The project ships two entry points:

- `gdproto`: a direct CLI for one-off generation from a `.proto` file.
- `protoc-gen-gdscript`: a standard `protoc` plugin that also works from
  tools such as Buf.

Both paths generate message wrapper classes that know how to serialize and
deserialize protobuf binary wire format. Generated code also includes a
round-trippable text format that is useful for debug output, fixtures, and
hand-edited test data.

## What Gets Generated

For each generated proto file, `gdproto` emits a wrapper class containing:

- Top-level and nested GDScript enums.
- Message classes with typed field accessors.
- Repeated field helpers.
- Map field helpers.
- Oneof discriminant enums and setters that clear the previous oneof member.
- `to_bytes()` and `from_bytes()` for binary wire format.
- `to_text()` and `from_text()` for gdproto's text format.
- `_to_string()` for Godot debug output.

The generated runtime file, `proto_core_utils.gd`, contains low-level protobuf
encoding helpers, decoding helpers, and shared parse error values. Keep it with
the generated wrappers in your Godot project.

## When To Use Each Entry Point

Use `buf` or `protoc` plugin mode when your project already has a proto module,
multiple schemas, imports, or CI generation. Plugin output uses `.pb.gd`
filenames and preserves proto-relative paths under the configured output
directory.

Use the direct `gdproto` CLI when you want to compile one schema to one
specific `.gd` path. The direct CLI writes exactly the output path you pass with
`--output`, plus a sibling `proto_core_utils.gd`.

## Compatibility

Generated binary output is wire-compatible with standard protoc decoders for
the supported proto3 feature set. Services, proto2 syntax, custom options,
extensions, JSON mapping, and first-class well-known type behavior are outside
the current scope.
