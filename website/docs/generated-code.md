---
title: Generated GDScript
description: Use generated message wrappers in Godot.
---

# Generated GDScript

`gdproto` emits **one `.pb.gd` file per top-level proto message or top-level
enum**. Each file extends `RefCounted` and declares a top-level `class_name`,
so the class is registered as a global identifier in Godot and can be used
directly — no `preload` needed.

Every generated wrapper depends on the sibling `proto_core_utils.gd`
(registered globally as `ProtoCoreUtils`), so the runtime file must stay in
the same output directory as the wrappers.

## File and Class Naming

The class prefix is derived from the input `.proto` filename by default —
`example.proto` becomes prefix `Example`. Given:

```protobuf
// example.proto
syntax = "proto3";

enum PlayerStatus { OFFLINE = 0; ONLINE = 1; AWAY = 2; IN_GAME = 3; }

message Player {
  string username = 1;
  message Position { float x = 1; float y = 2; float z = 3; }
  Position position = 7;
}

message GameState {
  repeated Player players = 1;
}
```

the generator writes:

| File | `class_name` | Holds |
| --- | --- | --- |
| `ExamplePlayer.pb.gd` | `ExamplePlayer` | The `Player` message. |
| `ExamplePlayerPosition.pb.gd` | `ExamplePlayerPosition` | The nested `Player.Position` message, flattened. |
| `ExampleGameState.pb.gd` | `ExampleGameState` | The `GameState` message. |
| `ExamplePlayerStatus.pb.gd` | `ExamplePlayerStatus` | A wrapper class holding `enum PlayerStatus { ... }`. |

Nested messages are flattened into siblings using the prefix plus the dotted
type path joined together (`Player.Position` -> `ExamplePlayerPosition`).

The matching golden files live under `examples/golden/` in the repository.

## Enum Addressing

Two rules cover every enum:

- **Nested enums** (declared inside a message) stay inline on the generated
  message class. If `Player` had `enum Status { OFFLINE = 0; ONLINE = 1; }`,
  values are accessed as `ExamplePlayer.Status.ONLINE`.
- **Top-level enums** get their own `<Prefix><EnumName>.pb.gd` wrapper class
  that contains the enum as an inner type. Values are accessed as
  `<Prefix><EnumName>.<EnumName>.<VALUE>`. For `PlayerStatus` in the example
  above:

  ```gdscript
  var s := ExamplePlayerStatus.PlayerStatus.ONLINE
  ```

  The extra `class_name` wrapper exists so top-level enum values stay
  globally addressable in Godot, which does not allow free-standing
  top-level enums in autoloaded scripts.

## Class Prefix

The default prefix comes from the input `.proto` path. Each path segment is
split on non-alphanumerics, PascalCased, and concatenated:

| Input path | Derived prefix |
|---|---|
| `example.proto` | `Example` |
| `game_state.proto` | `GameState` |
| `nested/foo_bar.proto` | `NestedFooBar` |
| `v1/api.proto` | `V1Api` |
| `uzir/common/v1/common.proto` | `UzirCommonV1Common` |

Using the entire path (not just the basename) keeps prefixes unique across
monorepo layouts that segregate otherwise-identical filenames into
different directories — without this, several `common.proto` files in
different packages would all derive to `Common` and collide at generation
time.

To override the default, use the `(gdproto.class_prefix)` file option:

```protobuf
syntax = "proto3";
import "gdproto/options.proto";

option (gdproto.class_prefix) = "Game";

message Hero {
  string name = 1;
  int32 hp = 2;
}
```

With the prefix above, the generator writes `GameHero.pb.gd` (class
`GameHero`).

The `import "gdproto/options.proto";` line is **required** for `protoc` and
`buf` because both tools reject unknown extensions. The direct `gdproto` CLI
tolerates a missing import, but importing it everywhere keeps a single
schema portable across all three paths.

There are three supported ways to install the options proto:

1. **Print it from the binary.** Simplest, no clone needed:

   ```bash
   mkdir -p proto/gdproto
   gdproto --print-options-proto > proto/gdproto/options.proto
   ```

   `protoc-gen-gdscript --print-options-proto` works the same way.

2. **Vendor and pass with `-I`** when using raw `protoc`:

   ```bash
   protoc \
     --plugin=protoc-gen-gdscript="$(which protoc-gen-gdscript)" \
     --gdscript_out=godot/generated \
     -I proto \
     -I path/to/vendored/gdproto \
     proto/example.proto
   ```

3. **Place inside the buf module** when using Buf — drop
   `gdproto/options.proto` under the path referenced by `modules:` in
   `buf.yaml` (see [Using buf](./buf.md#custom-class-prefix)).

`gdproto.class_prefix` uses field number `51000`. Protobuf reserves the
range `50000`-`99999` for internal third-party extensions; see
[custom options](https://protobuf.dev/programming-guides/proto3/#customoptions).

### Cross-File References Honor Imported Prefixes

When a generated wrapper references a message or enum defined in an
imported `.proto`, the rendered type uses the **imported file's**
`(gdproto.class_prefix)` — or its filename-derived prefix when the option
is absent. The importer's prefix is not applied to imported types. This
means a single project can mix files with explicit `class_prefix` options
and files that rely on the default, and cross-file references resolve to
the right class names in either direction.

## Construct A Message

```gdscript
var msg := ExamplePlayer.new()
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

`new_<field>()` creates a nested message instance. Note that the returned
value is an `ExamplePlayerPosition` — nested messages live in their own
sibling files but the field accessor name is unchanged.

## Oneofs

Each oneof group gets a generated enum ending in `OneOf`.

```gdscript
msg.set_email("alice@example.com")

if msg.has_email():
    print(msg.get_email())

match msg.get_contact_case():
    ExamplePlayer.ContactOneOf.EMAIL:
        print("email contact")
    ExamplePlayer.ContactOneOf.DISCORD:
        print("Discord contact")
    ExamplePlayer.ContactOneOf.UNSET:
        print("no contact")
```

Setting one member updates the oneof discriminant and clears the previous
member's value.

## Enums In Use

```gdscript
msg.set_status(ExamplePlayerStatus.PlayerStatus.ONLINE)

match msg.get_status():
    ExamplePlayerStatus.PlayerStatus.ONLINE:
        print("online")
    ExamplePlayerStatus.PlayerStatus.OFFLINE:
        print("offline")
```

Enum fields are stored as integers at runtime.

## Binary Round Trip

```gdscript
var bytes: PackedByteArray = msg.to_bytes()

var decoded := ExamplePlayer.new()
var err := decoded.from_bytes(bytes)
if err != ProtoCoreUtils.ProtobufError.NO_ERRORS:
    push_error("decode failed: %s" % err)
```

`to_bytes()` writes protobuf binary wire format for the supported proto3
feature set. `from_bytes()` returns a `ProtoCoreUtils.ProtobufError` value.

## Text Format Round Trip

```gdscript
var text: String = msg.to_text()

var copy := ExamplePlayer.new()
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
