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

For a schema that follows [protobuf's own style
guide](https://protobuf.dev/programming-guides/style/), generated output passes
[gdkit](https://github.com/cafecito-games/gdkit)'s `gdkit format check`,
`gdkit lint check`, and `gdkit uid check` using gdkit's default configuration. A
project that gates CI on gdkit can keep its generated protocol directory inside
the checked set rather than excluding the directory — and with it, the checks
that would catch a real problem.

That is the whole guarantee, and its edges are worth reading before you rely on
it: identifier spelling has to follow the style guide, line length is corpus
dependent, and the agreement is with gdkit's *default* configuration. See
[Known Limits](#known-limits).

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

Every generated `.pb.gd` file opens with this comment on line 1:

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
The sibling `proto_core_utils.gd` runtime carries no such directive, and needs
none: it is a fixed file rather than one generated per message, so its length
and its method and return counts do not grow with the schema, and it stays
within every default limit on its own.


### `uid://` Sidecars

Every generated `X.gd` is written with a sibling `X.gd.uid` holding one line:

```text
uid://ix8k3hu6vdsf
```

Godot assigns these resource identifiers at random (`ResourceUID::create_id`)
the first time it imports a script. gdproto derives the identifier from the
filename instead, which has three benefits:

- Regenerating a schema produces byte-identical output, sidecars included, so
  generated files stay clean in version control.
- Two developers generating the same schema get the same identifiers.
- It is the only scheme that works in the `protoc` plugin path, which never
  learns its own output directory and so cannot read identifiers already on
  disk.

#### The Contract

**A generated script is addressed by its `class_name`, not by its `uid://`
path.** Every generated file declares a `class_name`, which registers the class
as a global identifier in Godot, and generated messages are instantiated
through that name:

```gdscript
var player := ExamplePlayer.new()
```

Nothing should reference a generated script by `uid://` — not a scene, not a
resource, not `project.godot`. A generated file is compiler output; it is reached
by name, the way a library class is, and never by resource path.

That contract is what makes a derived identifier sufficient. The identifier only
has to be *stable*, so that regenerating a schema does not churn the working
tree, and deriving it from the filename guarantees exactly that. It does not
have to be preserved across a change of scheme, because nothing resolves it.

A project that happens to hold Godot-assigned sidecars for generated scripts
will therefore see them replaced the first time it regenerates with gdproto, and
that is inconsequential: the identifiers being replaced were not referenced by
anything. In a production Godot client with 688 generated scripts, none of the
689 generated sidecar identifiers appeared in any scene, resource, or project
file.

### Known Limits

The conformance guarantee has three real edges.

**Identifier spelling has to follow protobuf's style guide.** Proto identifiers
are carried through into GDScript names verbatim — the generator does not rename
them — so a schema whose identifiers protobuf permits but gdkit's naming rules
reject produces lint diagnostics. This schema is legal proto3:

```protobuf
message Account {
  string userName = 1;
}
```

and its output reports three diagnostics, because the field becomes `var
_userName` with `get_userName()` and `set_userName()` accessors:

```text
Error: Class-scope variable name "_userName" is not valid (class-variable-name)
Error: Function name "set_userName" is not valid (function-name)
Error: Function name "get_userName" is not valid (function-name)
```

The same applies to type and enum spelling: a `snake_case` message name reports
`class-name`, a `snake_case` enum name reports `enum-name`, and a `camelCase`
enum value reports `enum-element-name`.

The [protobuf style
guide](https://protobuf.dev/programming-guides/style/) already mandates
`lower_snake_case` field names, `PascalCase` message and enum names, and
`SCREAMING_SNAKE_CASE` enum values, so a conforming schema — the normal case —
runs into none of this.

These naming rules are deliberately **not** suppressed. They are real naming
rules, and silencing them in generated output would hide genuine naming problems
there. The alternative, renaming proto identifiers on the way into GDScript,
would change the generated API — a deliberate decision this project has not
taken, because a field's proto name is how a schema author expects to address it.
Two remedies are available: rename the offending fields in the schema, which the
protobuf style guide already asks for, or scope gdkit's naming rules away from
the generated directory in the consuming project's own `.gdkit/lint.json`.

**`max-line-length` reports for deep package paths, and that is the common
case.** Class prefixes are derived from the full `.proto` path, so a layout
like `uzir/assetpack/client/v1/characters.proto` produces class names near 50
characters. A declaration that names its type twice then cannot fit in 100
columns, and has no legal wrap point:

```gdscript
var msg_instance: UzirAssetpackClientV1CharactersAvatarBodyDirection = UzirAssetpackClientV1CharactersAvatarBodyDirection.new()
```

Regenerating a 687-file production schema produces 868 of these and no other
lint diagnostic, while `format check` and `uid check` pass that same schema
completely. Treat it as expected for a monorepo layout rather than as an edge
case. Shortening the prefix with `(gdproto.class_prefix)` reduces it; inferring
the type would remove it but breaks under a strict `untyped_declaration`
warning policy, so the generator keeps the annotation. A project that wants the
generated directory silent on this disables or allowlists the one rule for that
directory.

This rule is deliberately left reporting rather than suppressed alongside
`max-file-lines`, and the asymmetry is the point. Wrapping is genuinely the
formatter's job, and only the one case it cannot touch — a single over-long
identifier or type annotation with no legal wrap point — escapes it, so a
`max-line-length` diagnostic in generated output is worth seeing and
suppressing the rule would hide real formatting problems. A file-length
diagnostic carries no such signal, because nothing the generator could do
would make it go away.

**An opt-in logging rule reports.** Generated code calls `push_error` to report
a decode failure, so a project that enables gdkit's `no-engine-logging` rule —
inert by default — and routes diagnostics through its own logger will see that
rule report across generated files. The generator cannot know the project's
logger, so scope the rule away from the generated directory.

**The agreement is with gdkit's default configuration.** gdproto cannot read a
downstream project's `.gdkit/format.json`, so a project that customizes
`line_width`, indents with spaces, or otherwise diverges from gdparser's Godot
style needs its own `gdkit format write` pass over the generated directory
after generation.

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
