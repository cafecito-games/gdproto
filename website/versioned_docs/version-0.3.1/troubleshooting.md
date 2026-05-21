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

Keep the runtime file with the generated wrappers. Plugin mode emits
`proto_core_utils.gd` at the output root when files are generated. Direct CLI
mode writes it next to the requested output file.

Do not move generated wrappers without moving the runtime to the path expected
by the generated preload.

## Imported Message Class Is Missing In Godot

If a proto file imports another message type, generate wrappers for the imported
proto files too. Generated GDScript references imported messages by wrapper
class name.

Buf is the easiest way to keep the generated set complete because it supplies
the plugin with descriptors for the input module.

## Output File Name Is Different Between CLI And Plugin Mode

Direct CLI mode writes the exact path supplied with `--output`:

```bash
gdproto proto/player.proto -o godot/generated/player.gd
```

Plugin mode computes `.pb.gd` filenames from proto-relative paths:

```bash
protoc --gdscript_out=godot/generated -I proto proto/player.proto
```

That writes `godot/generated/player.pb.gd`.

## Validation Fails For A Schema Feature

Check [Feature support](./feature-support.md). gdproto intentionally rejects or
ignores features that do not map to generated Godot data wrappers, including
proto2 syntax, services, extensions, and protobuf JSON mapping helpers.
