---
title: Troubleshooting
description: Common generation and Godot import issues.
---

# Troubleshooting

## `protoc-gen-gdscript: program not found`

Install the plugin and make sure the install directory is on `PATH`:

```bash
brew install --cask cafecito-games/tap/gdproto
which protoc-gen-gdscript
```

Or install with Go:

```bash
go install github.com/cafecito-games/gdproto/cmd/protoc-gen-gdscript@latest
export PATH="$(go env GOPATH)/bin:$PATH"
which protoc-gen-gdscript
```

With `protoc`, you can also pass the binary explicitly:

```bash
protoc \
  --plugin=protoc-gen-gdscript=/absolute/path/to/protoc-gen-gdscript \
  --gdscript_out=godot/generated \
  -I proto \
  proto/player.proto
```

## Buf Cannot Find The Plugin

For this `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

Buf runs `protoc-gen-gdscript` from the environment. Run `which
protoc-gen-gdscript` in the same shell or CI step that runs `buf generate`.

## Generated Godot Script Cannot Preload `proto_core_utils.gd`

Keep the runtime file with the generated wrappers. Both the direct CLI and
the plugin emit `proto_core_utils.gd` at the output directory root alongside
the generated `.pb.gd` files.

Do not move generated wrappers without moving the runtime to the path expected
by the generated preload.

## Imported Message Class Is Missing In Godot

The plugin only generates `.pb.gd` files for the protos explicitly listed in
`file_to_generate` (the files you pass on the command line or include in the
buf module). Imported `.proto` files are parsed for type resolution but do
not produce wrappers automatically.

If a generated wrapper references an imported message type, add the
imported `.proto` to the input set in the same invocation so its wrapper
is generated too.

## `-o` Rejected With "Looks Like A File"

The direct CLI's `-o` flag takes an **output directory**, not a file path.
It rejects values that end in `.gd` or point at an existing file. Pass a
directory instead:

```bash
gdproto proto/example.proto -o godot/generated/
```

This writes one `.pb.gd` per top-level message or enum (for example
`ExamplePlayer.pb.gd`, `ExampleGameState.pb.gd`) plus
`proto_core_utils.gd`.

## `(gdproto.class_prefix)` Is Not Applied

Both `protoc` and `buf` require the `gdproto/options.proto` extension
descriptor to be reachable from your proto sources. The schema must
`import "gdproto/options.proto";` and the descriptor must live on an `-I`
import root (raw `protoc` or the direct `gdproto` CLI) or inside the buf
module (Buf). Vendor it with:

```bash
mkdir -p proto-include/gdproto
gdproto --print-options-proto > proto-include/gdproto/options.proto
```

Then pass the include root when invoking the direct CLI (the flag is
repeatable and matches `protoc`'s convention):

```bash
gdproto -I proto-include -o godot/generated proto/example.proto
```

The direct `gdproto` CLI tolerates a missing import, but importing it
everywhere keeps a single schema portable. See
[Generated GDScript](./generated-code.md#class-prefix).

## `import "X" not found from Y`

The direct CLI prints this when an import cannot be resolved. Add the
directory that contains the imported `.proto` as an `-I/--proto_path`
include root. For vendored extension protos like
`gdproto/options.proto`, that means the directory whose child
`gdproto/options.proto` exists:

```bash
gdproto -I proto-include -o godot/generated proto/example.proto
```

The flag is repeatable; each include root is searched in order before the
input file's own directory is consulted.

## Validation Fails For A Schema Feature

Check [Feature support](./feature-support.md). gdproto intentionally rejects or
ignores features that do not map to generated Godot data wrappers, including
proto2 syntax, services, extensions, and protobuf JSON mapping helpers.
