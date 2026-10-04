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

Each generated script is accompanied by a `.uid` sidecar carrying its Godot
resource identifier, and the output as a whole is written in a form that
passes gdkit's checks — see [gdkit conformance](#gdkit-conformance).

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

## gdkit Conformance

Generated output passes [gdkit](https://github.com/cafecito-games/gdkit)'s
`gdkit format check`, `gdkit lint check`, and `gdkit uid check` using gdkit's
default configuration. A project that gates CI on gdkit can keep its generated
protocol directory inside the checked set rather than excluding the directory —
and with it, the checks that would catch a real problem.

A test in the repository generates `examples/example.proto` and the map-heavy
`tests/godot/fixtures/proto/collections.proto` into throwaway Godot projects and
runs gdkit's linter, formatter, and uid check over each in process, so a
generator change that breaks conformance fails the gdproto test suite instead of
surfacing in a downstream project. The second schema is there because its output
runs well past a thousand lines, which is the only way a file-length limit gets
exercised at all.

### Formatting

The generator parses its own output with
[gdparser](https://github.com/cafecito-games/gdparser) and re-emits it through
`gdparser/format` with the Godot style options. `gdkit format` delegates to that
same engine, and gdkit's default format configuration is field for field
gdparser's `GodotStyle()`, so the two agree by construction instead of gdproto
reimplementing gdkit's wrapping rules. The generator additionally checks that a
second formatting pass changes nothing, because `gdkit format check` reports any
file another pass would rewrite.

Because the agreement runs through a pinned library version, formatting matches
the gdkit release gdproto was built against; `go.mod` pins gdkit and gdparser
together for that reason. A much newer `gdkit` binary could in principle format
a construct differently, which a `gdkit format write` pass over the generated
directory resolves.

### Definition Order

Oneof discriminant enums are emitted ahead of the field `var` declarations,
because gdkit's `class-definitions-order` rule places the enums slot before the
var slots.

### The `gdkit:disable` Directive

Every generated file opens with this comment on line 1:

```gdscript
# gdkit:disable = max-returns, max-public-methods, max-file-lines
```

All three suppressed rules are gdkit *design limits* whose value scales with the
schema rather than with the quality of the code, so generated protobuf code
cannot satisfy them by construction:

- `max-returns` — a wire-format parser is an early-return function with one
  return per field.
- `max-public-methods` — every field contributes a set of public accessors
  (`get_`, `set_`, `has_`, `clear_`, and more for repeated, map, and message
  fields), so method count grows with the schema.
- `max-file-lines` — a message's file is as long as its field count demands:
  serializer, deserializer, text-format reader and writer, accessors, and enum
  helpers are all emitted per field. The repository's own `collections.proto`
  fixture generates 1,121 lines from a single map-heavy message, against a
  default limit of 1,000, and no generator change brings a large message under a
  fixed line count.

Restructuring generated code to fit those limits would make it worse to read,
so they are suppressed rather than worked around. **No other rule is
suppressed**; everything else gdkit's default lint configuration checks is
satisfied outright. The directive has to be the first line because
`max-public-methods` is reported against the class global scope there, and a
`gdkit:disable` applies only from its own line to the end of the file.

### `uid://` Sidecars

Every generated `X.gd` is written with a sibling `X.gd.uid` holding one line:

```text
uid://ix8k3hu6vdsf
```

Godot assigns these resource identifiers at random (`ResourceUID::create_id`)
the first time it imports a script. gdproto derives the identifier from the
filename instead. The reason is regeneration: a generator re-runs on every
schema sync, and reissuing an identifier changes what every existing `uid://`
reference to that script resolves to, with nothing in the project rewriting
those references. Deriving it has three further benefits:

- Regenerating a schema produces byte-identical output, sidecars included, so
  generated files stay clean in version control.
- Two developers generating the same schema get the same identifiers.
- It is the only scheme that works in the `protoc` plugin path, which never
  learns its own output directory and so cannot read identifiers already on
  disk.

### Known Limits

The conformance guarantee has three real edges.

**`max-line-length` is corpus dependent.** A schema with very long message,
field, or enum names still produces lines past gdkit's 100-column limit: a
single long identifier or type annotation has no legal wrap point, so no
formatter can bring the line down. A deliberately long-named probe schema
produces 11 such diagnostics with a longest line of 133 columns. Formatting
stays canonical and idempotent for those files — it is only the line-length
lint rule that reports. A project in that situation can allowlist the rule for
the generated directory, or shorten the schema's names.

This rule is deliberately left reporting rather than suppressed alongside
`max-file-lines`, and the asymmetry is the point. Wrapping is genuinely the
formatter's job, and only the one case it cannot touch — a single over-long
identifier or type annotation with no legal wrap point — escapes it, so a
`max-line-length` diagnostic in generated output is worth seeing and
suppressing the rule would hide real formatting problems. A file-length
diagnostic carries no such signal, because nothing the generator could do
would make it go away.

**The agreement is with gdkit's default configuration.** gdproto cannot read a
downstream project's `.gdkit/format.json`, so a project that customizes
`line_width`, indents with spaces, or otherwise diverges from gdparser's Godot
style needs its own `gdkit format write` pass over the generated directory
after generation.

**The first regeneration replaces Godot-assigned sidecars.** A project that
already contains `.uid` files Godot wrote for generated scripts will see them
replaced on the first regeneration with gdproto, because the derived identifier
differs from the random one Godot assigned. The exposure is usually nil: every
generated class declares `class_name` and is referenced through that global
identifier, not through a `uid://` path. But it is a real one-time event, so if
a scene or resource does reference a generated script by uid, grep the project
for the old identifiers and update those references.

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
