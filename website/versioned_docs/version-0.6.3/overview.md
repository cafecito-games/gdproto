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

`gdproto` emits **one `.pb.gd` file per top-level proto message or top-level
enum**. Each file declares a `class_name` derived from a per-file class prefix
plus the proto type name; nested messages are flattened into siblings using
the same prefix. For example, given an `example.proto` containing a `Player`
message with a nested `Position` and a top-level `PlayerStatus` enum, the
generator writes:

```text
ExamplePlayer.pb.gd          # class_name ExamplePlayer
ExamplePlayerPosition.pb.gd  # class_name ExamplePlayerPosition (flattened nested)
ExampleGameState.pb.gd
ExamplePlayerStatus.pb.gd    # class_name ExamplePlayerStatus, contains `enum PlayerStatus`
proto_core_utils.gd
```

Each generated message wrapper supports:

- Typed field accessors for scalars, messages, repeated, and map fields.
- Oneof discriminant enums and setters that clear the previous oneof member.
- `to_bytes()` and `from_bytes()` for binary wire format.
- `to_text()` and `from_text()` for gdproto's text format.
- `_to_string()` for Godot debug output.

Nested enums (declared inside a message) stay inline on the message class;
top-level enums get a thin wrapper class so the values are still globally
addressable in Godot. See [Generated GDScript](./generated-code.md) for the
full addressing rules.

The class prefix is derived from the input `.proto` filename by default. To
override it, set the `(gdproto.class_prefix)` file option — see
[Generated GDScript](./generated-code.md#class-prefix) for the option and the
three supported ways to install `gdproto/options.proto`.

The generated runtime file, `proto_core_utils.gd`, contains low-level protobuf
encoding helpers, decoding helpers, and shared parse error values. Keep it with
the generated wrappers in your Godot project.

## When To Use Each Entry Point

Use `buf` or `protoc` plugin mode when your project already has a proto module,
multiple schemas, imports, or CI generation. Plugin output preserves
proto-relative directories under the configured output directory and emits
one `.pb.gd` file per top-level type.

Use the direct `gdproto` CLI when you want to compile one schema into an
output directory without setting up `protoc` or Buf. The direct CLI's `-o`
flag takes an **output directory**; it writes one `.pb.gd` per top-level type
plus a sibling `proto_core_utils.gd`.

## Compatibility

Generated binary output is wire-compatible with standard protoc decoders for
the supported proto3 feature set. Services, proto2 syntax, custom options,
extensions, JSON mapping, and first-class well-known type behavior are outside
the current scope.
