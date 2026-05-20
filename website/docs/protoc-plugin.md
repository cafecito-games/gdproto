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
  proto/player.proto
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
  proto/player.proto
```

## Output

Plugin output uses proto-relative paths and snake-cased `.pb.gd` filenames:

| Proto path | Generated wrapper |
| --- | --- |
| `player.proto` | `player.pb.gd` |
| `PlayerStats.proto` | `player_stats.pb.gd` |
| `google/protobuf/timestamp.proto` | `google/protobuf/timestamp.pb.gd` |

When files are generated, the plugin also emits:

```text
proto_core_utils.gd
```

The runtime file is written at the plugin output root. Generated wrappers
preload it as a sibling runtime, so keep your generated wrappers and runtime in
the layout produced by the plugin.

## Imports

Set every import root with `-I`:

```bash
protoc \
  --gdscript_out=godot/generated \
  -I proto \
  -I third_party/proto \
  proto/player.proto
```

If a generated wrapper references an imported message type, the imported proto
also needs a generated GDScript wrapper. Prefer one `protoc` invocation that
includes all files you want available in Godot, or use Buf to keep the input set
consistent.

## Errors

Plugin validation errors are reported through protoc's plugin response. If
generation fails, check:

- The schema uses `syntax = "proto3";`.
- Import paths are reachable from the supplied `-I` roots.
- The schema avoids unsupported features such as services, extensions, proto2,
  and custom options.
