---
title: protoc plugin
description: Run protoc-gen-gdscript directly from protoc.
---

# protoc plugin

`protoc-gen-gdscript` implements the standard protoc plugin protocol. Use this
mode when you already drive generation with `protoc` or want to mirror what Buf
does internally.

## Command

```bash
protoc \
  --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
  --gdscript_out=godot/generated \
  -I proto \
  proto/example.proto
```

The plugin name is derived from the output flag:

- `--gdscript_out=...` asks `protoc` to run `protoc-gen-gdscript`.
- `--plugin=protoc-gen-gdscript=...` tells `protoc` exactly which binary to
  run.

If `protoc-gen-gdscript` is already on `PATH`, the explicit `--plugin` flag is
optional:

```bash
protoc \
  --gdscript_out=godot/generated \
  -I proto \
  proto/example.proto
```

## Output

The plugin writes **one `.pb.gd` file per top-level message or top-level
enum** in each input proto. Nested messages are flattened into sibling files
using the same class prefix. Files are placed under the configured output
directory; if the input file lives in a subdirectory of an import root,
that subdirectory is preserved.

For `proto/example.proto` from the project examples, the plugin writes:

```text
godot/generated/
  ExamplePlayer.pb.gd
  ExamplePlayerPosition.pb.gd
  ExampleGameState.pb.gd
  ExamplePlayerStatus.pb.gd
  proto_core_utils.gd
```

The runtime file is written at the plugin output root. Generated wrappers
preload it as a sibling runtime, so keep your generated wrappers and runtime in
the layout produced by the plugin.

## Files Generated

The plugin only emits output for files explicitly listed in
`file_to_generate` (the files you pass on the `protoc` command line). This
matches `protoc-gen-go` and other standard plugins. Imported `.proto` files
are still parsed for type resolution, but they do **not** produce
`.pb.gd` files automatically.

If you need wrappers for an imported `.proto`, add it to the input set in
the same `protoc` invocation, or run a second `protoc` command for it.

Cross-file type references render with the imported file's own
`(gdproto.class_prefix)` (or its filename-derived default) — the importer's
prefix is not applied to imported types. See [Generated GDScript](./generated-code.md#cross-file-references-honor-imported-prefixes).

## Imports

Set every import root with `-I`:

```bash
protoc \
  --gdscript_out=godot/generated \
  -I proto \
  -I third_party/proto \
  proto/example.proto
```

## Custom Class Prefix

The default class prefix for each generated file is derived from the input
`.proto` filename. To override it, add the `(gdproto.class_prefix)` file
option to your schema:

```protobuf
syntax = "proto3";
import "gdproto/options.proto";

option (gdproto.class_prefix) = "Game";

message Hero {
  string name = 1;
}
```

`protoc` requires the descriptor for the `gdproto.class_prefix` extension to
be available at parse time. Vendor it with one of:

```bash
mkdir -p proto/gdproto
gdproto --print-options-proto > proto/gdproto/options.proto
# or
protoc-gen-gdscript --print-options-proto > proto/gdproto/options.proto
```

Then make sure the directory containing `gdproto/options.proto` is on a
`-I` import root:

```bash
protoc \
  --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
  --gdscript_out=godot/generated \
  -I proto \
  proto/example.proto
```

See [Generated GDScript](./generated-code.md#class-prefix) for the full
explanation of how the prefix affects class names.

## Errors

Plugin validation errors are reported through protoc's plugin response. If
generation fails, check:

- The schema uses `syntax = "proto3";`.
- Import paths are reachable from the supplied `-I` roots.
- If the schema uses `(gdproto.class_prefix)`, `gdproto/options.proto` is on
  an `-I` root and imported from the schema.
- The schema avoids unsupported features such as services, extensions, and
  proto2.
