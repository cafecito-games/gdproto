---
title: Using buf
description: Configure buf generate to run protoc-gen-gdscript.
---

# Using buf

Buf can run `protoc-gen-gdscript` as a local plugin. The plugin binary must be
installed on `PATH` before you run `buf generate`.

## Example Layout

```text
game/
  buf.yaml
  buf.gen.yaml
  proto/
    player.proto
  godot/
    generated/
```

## buf.yaml

For a single local proto module:

```yaml
version: v2
modules:
  - path: proto
```

## buf.gen.yaml

Use a local plugin entry:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

Buf resolves `local: protoc-gen-gdscript` the same way your shell does. Verify
the binary before generation:

```bash
which protoc-gen-gdscript
```

Then run:

```bash
buf generate
```

Buf reads `buf.gen.yaml`, builds a CodeGeneratorRequest for your input module,
and invokes the plugin.

## Output Paths

Plugin mode preserves the proto-relative path below the configured output
directory and uses `.pb.gd` filenames.

With this input:

```text
proto/player.proto
```

and this config:

```yaml
version: v2
plugins:
  - local: protoc-gen-gdscript
    out: godot/generated
```

the generated wrapper is:

```text
godot/generated/player.pb.gd
```

The plugin also emits:

```text
godot/generated/proto_core_utils.gd
```

Keep `proto_core_utils.gd` with the generated wrappers in your Godot project.

## Imports And Generated Wrappers

Generated GDScript references imported message types through their generated
wrapper classes. If `player.proto` imports `shared/team.proto`, the imported
schema needs a generated wrapper too.

Buf supplies transitive descriptors to plugins, and `protoc-gen-gdscript`
generates wrappers for imported files that are present in the request. This is
important for Godot class resolution when a generated file references another
generated proto wrapper.

## CI Example

```yaml
name: Generate protos

on:
  push:
    branches: [main]

jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26.x"

      - uses: bufbuild/buf-action@v1

      - name: Install gdproto plugin
        run: go install github.com/cafecito-games/gdproto/cmd/protoc-gen-gdscript@latest

      - name: Generate GDScript
        run: buf generate
```

Pin the plugin version in CI when reproducible generation matters.
