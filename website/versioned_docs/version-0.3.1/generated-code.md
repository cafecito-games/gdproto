---
title: Generated GDScript
description: Use generated message wrappers in Godot.
---

# Generated GDScript

Generated wrappers are plain GDScript classes that extend `RefCounted`. Each
file declares a `class_name`, so the wrapper (e.g. `PlayerProto` from
`player.pb.gd`) is registered as a global identifier and can be used directly.

The wrapper depends on the sibling `proto_core_utils.gd` (registered globally as
`ProtoCoreUtils`), so the runtime file must stay in the generated output tree.

## Construct A Message

```gdscript
var msg := PlayerProto.Player.new()
msg.set_username("alice")
msg.set_level(42)
```

Scalar fields get `set_<field>()`, `get_<field>()`, `has_<field>()`, and
`clear_<field>()` methods.

## Repeated Fields

```gdscript
msg.add_inventory("sword")
msg.add_inventory("potion")

for item in msg.get_inventory():
    print(item)
```

Repeated field helpers expose append-style methods for generation-safe access.

## Maps

```gdscript
msg.add_stats("strength", 100)

if msg.get_stats().has("strength"):
    print(msg.get_stats()["strength"])
```

Map fields use Godot dictionaries internally and expose key-based helpers.

## Nested Messages

```gdscript
var pos := msg.new_position()
pos.set_x(1.0)
pos.set_y(2.0)
pos.set_z(3.0)
```

`new_<field>()` creates a nested message instance for message-typed fields.

## Oneofs

Each oneof group gets a generated enum ending in `OneOf`.

```gdscript
msg.set_email("alice@example.com")

if msg.has_email():
    print(msg.get_email())

match msg.get_contact_case():
    PlayerProto.Player.ContactOneOf.EMAIL:
        print("email contact")
    PlayerProto.Player.ContactOneOf.DISCORD:
        print("Discord contact")
    PlayerProto.Player.ContactOneOf.UNSET:
        print("no contact")
```

Setting one member updates the oneof discriminant and clears the previous
member's value.

## Enums

Top-level proto enums become top-level GDScript enums. Nested proto enums stay
nested under their generated message class.

```gdscript
msg.set_status(PlayerProto.PlayerStatus.ONLINE)

match msg.get_status():
    PlayerProto.PlayerStatus.ONLINE:
        print("online")
    PlayerProto.PlayerStatus.OFFLINE:
        print("offline")
```

Enum fields are stored as integers at runtime.

## Binary Round Trip

```gdscript
var bytes: PackedByteArray = msg.to_bytes()

var decoded := PlayerProto.Player.new()
var err := decoded.from_bytes(bytes)
if err != ProtoCoreUtils.ProtobufError.NO_ERRORS:
    push_error("decode failed: %s" % err)
```

`to_bytes()` writes protobuf binary wire format for the supported proto3
feature set. `from_bytes()` returns a `ProtoCoreUtils.ProtobufError` value.

## Text Format Round Trip

```gdscript
var text: String = msg.to_text()

var copy := PlayerProto.Player.new()
var err := copy.from_text(text)
if err != ProtoCoreUtils.ProtobufError.NO_ERRORS:
    push_error("text decode failed: %s" % err)
```

The text format is designed for gdproto round trips and debug visibility. Use
binary wire format for compatibility with other protobuf runtimes.

## Debug Output

```gdscript
print(msg)
```

Generated messages implement `_to_string()` using the text format.
