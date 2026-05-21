---
title: Quickstart
description: Generate GDScript from a proto file and use it in Godot.
---

# Quickstart

This quickstart uses the `protoc-gen-gdscript` plugin because it is the path
you will normally use from Buf, `protoc`, or CI.

## 1. Install The Plugin

With Homebrew:

```bash
brew install --cask cafecito-games/tap/gdproto
```

Or with Go:

```bash
go install github.com/cafecito-games/gdproto/cmd/gdproto@latest
go install github.com/cafecito-games/gdproto/cmd/protoc-gen-gdscript@latest
```

Both paths install `gdproto` and `protoc-gen-gdscript`. If you use Go, make sure
`$GOPATH/bin` is on `PATH`:

```bash
which protoc-gen-gdscript
```

## 2. Add A Proto Schema

```protobuf
syntax = "proto3";

package game;

message Player {
  string username = 1;
  uint32 level = 2;
  repeated string inventory = 3;
}
```

Save it as `proto/player.proto`.

## 3. Generate GDScript

With `protoc`:

```bash
mkdir -p godot/generated
protoc \
  --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
  --gdscript_out=godot/generated \
  -I proto \
  proto/player.proto
```

The plugin writes:

```text
godot/generated/
  player.pb.gd
  proto_core_utils.gd
```

With Buf, the same generation can be kept in `buf.gen.yaml`. See
[Using buf](./buf.md) for the full setup.

## 4. Use The Generated Code

Copy or generate the output into your Godot project. Each generated file declares
a `class_name`, so the wrapper (e.g. `PlayerProto`) is available as a global
identifier — no `preload` needed:

```gdscript
var player := PlayerProto.Player.new()
player.set_username("alice")
player.set_level(42)
player.add_inventory("sword")

var bytes: PackedByteArray = player.to_bytes()

var decoded := PlayerProto.Player.new()
var err := decoded.from_bytes(bytes)
if err != ProtoCoreUtils.ProtobufError.NO_ERRORS:
    push_error("decode failed: %s" % err)
```

`player.pb.gd` depends on the sibling `proto_core_utils.gd`, so keep both files
in the generated output directory.
