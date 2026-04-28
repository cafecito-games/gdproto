# gdproto 0.1.0 — Bug Report (from Uzir integration)

Found while integrating gdproto into the Uzir monorepo to generate
GDScript bindings for the netcode protobuf schema. All four issues are
**blockers** for using the generated output in a real Godot project.

Versions in play:

- gdproto: `0.1.0` (installed via `uv tool install` from `~/foss/gdproto`)
- protoc: `libprotoc 33.4` (Homebrew, macOS arm64)
- Godot: `4.5.stable.official.876b29033`

The schema lives in a buf-managed monorepo under `protocol/protocol/uzir/`
with packages `uzir.common.v1`, `uzir.assetpack.v1`, `uzir.assetpack.client.v1`,
`uzir.assetpack.server.v1`, `uzir.netcode.auth.v1`, `uzir.netcode.game.v1`,
`uzir.netcode.transport.v1`. ~40 `.proto` files, cross-package imports,
`oneof`, `service` blocks, `repeated`, `map`. No exotic features.

---

## Bug 1 — Parser rejects top-level `service { ... }` blocks

### Symptom

```
$ gdproto auth.proto -o auth.gd
auth.proto:52:1: error: Unexpected token: SERVICE
```

### Reproduction

```protobuf
syntax = "proto3";
package demo.v1;

message LoginRequest { string username = 1; }
message LoginResponse { string token = 1; }

service AuthService {
  rpc Login(LoginRequest) returns (LoginResponse);
}
```

`gdproto demo.proto -o demo.gd` fails on the `service` line.

### Expected

Either silently consume-and-discard the `service` block (gdproto generates
no RPC stubs anyway — verified by inspecting plugin-mode output for files
that contain services), or accept and skip it with an info-level note.

### Why this matters

`service` is part of standard proto3. Any schema used with gRPC will have
service blocks. Refusing to parse them means gdproto can only consume
schemas explicitly stripped for it — a hard fork the consumer must
maintain. We currently maintain such a fork at
`protocol/godot-protocol/`; we want to delete it.

### Source pointers

- `src/gdproto/lexer.py:30` already has `SERVICE = auto()` and
  `src/gdproto/lexer.py:94` maps `"service" → TokenType.SERVICE`.
- `src/gdproto/validator.py:119` also knows `"service"` is reserved.
- `src/gdproto/parser.py` is missing the production rule that handles
  the SERVICE token at top level. Adding `_parse_service_block(self)` that
  consumes through the matching `}` (with brace depth tracking for any
  nested options) and returns `None` (or a discarded AST node) should
  unblock everything.

### Test case to add

Lift `examples/buf/proto/game.proto` (or a stripped variant) into the
test fixtures with a `service` block; assert generation succeeds and
output has no service-related code.

---

## Bug 2 — Parser rejects reserved words as field names

### Symptom

```
$ gdproto game.proto -o game.gd
game.proto:43:10: error: Expected IDENTIFIER, got MESSAGE

$ gdproto dialogue.proto -o dialogue.gd
dialogue.proto:18:39: error: Expected IDENTIFIER, got OPTION
```

### Reproduction

```protobuf
syntax = "proto3";
package demo.v1;

message ActionRejected {
  string reason = 1;
  string message = 2;   // BUG: rejected
}

message SelectAction {
  string option = 1;    // BUG: rejected
}
```

### Expected

In proto3, the token immediately after the field type accepts any
identifier — including any reserved word in the language. `protoc`
accepts `message`, `option`, `service`, `enum`, `package`, `import`,
`syntax`, etc. as field names; `protoc-gen-go` produces compilable Go
code (e.g., `GetMessage()`).

### Why this matters

Renaming canonical fields to dodge this bug breaks API names for all
consumers (Go, JS, anywhere we have generated bindings) and trips
`buf breaking` checks. We had to rename `ActionRejected.message` →
`text` and `DialogueOption.option` → `choice` in our canonical schema
because gdproto refused them; both renames are bug workarounds, not
intentional design.

### Suggested fix

In the field-declaration rule, after parsing a type, accept any token
whose lexeme is a valid identifier — promote keyword tokens to identifier
tokens contextually rather than requiring strict `TokenType.IDENT`.

In gdproto, `set_<field>` / `get_<field>` will also need to disambiguate
when the field name happens to match a GDScript keyword (e.g., `class`,
`extends`); that's a smaller follow-up and can be handled with a
prefix/escape strategy.

### Test cases to add

A fixture `reserved_field_names.proto` declaring fields named:
`message`, `option`, `service`, `enum`, `package`, `import`, `syntax`,
`extend`, `oneof`, `map`, `repeated`, `reserved`, `returns`, `rpc`. The
generated GDScript should compile in Godot 4.5 (parse-test under
`godot --headless --check-only` or equivalent).

---

## Bug 3 — Standalone CLI does not resolve cross-file imports

### Symptom

After `gdproto world.proto -o world.gd`, the generated file contains:

```gdscript
class MapObject extends RefCounted:
    var _tile: uzir.common.v1.Tile = null
    var _definition_ref: uzir.assetpack.v1.ObjectDefRef = null
    ...
    func new_tile() -> uzir.common.v1.Tile:
        _tile = uzir.common.v1.Tile.new()
```

`uzir.common.v1.Tile` is **not valid GDScript syntax**. Godot 4.5 errors
out at parse time:

```
ERROR: Unexpected '.' in class body. (line 28)
```

### Reproduction

A `.proto` with `import "other_file.proto";` and a field whose type comes
from `other_file.proto`. Run `gdproto via the CLI with just the importing
file. The generated output uses `dotted.package.Type` syntax for the
foreign type.

### Expected

Either:

1. CLI mode emits `WrapperName.Type` syntax matching plugin mode (which
   already works correctly for cross-file references via the
   `_dependencies` map populated by `DescriptorConverter`), with the
   `WrapperName` derived deterministically from the source file path —
   plus a `const WrapperName = preload("res://path/to/other_file.pb.gd")`
   header line, OR
2. CLI mode is documented as "single-file only — use protoc plugin for
   multi-file projects" and the FQN fallback is removed (replaced with a
   hard error pointing the user to plugin mode).

### Source pointer

- `src/gdproto/generator/gdscript.py:241-251`
  `_get_qualified_type_name` falls through to `get_gdscript_type(field_type)`
  when `self._dependencies.get(source_file)` returns nothing; in CLI mode
  that map is never populated, so the fallback emits the FQN. The
  fallback's own comment says `# Fallback to regular type name (shouldn't happen)`.

### Why this matters

Any non-trivial schema has cross-file imports. CLI mode silently emits
broken GDScript; consumers think things work until they try to load the
output in Godot.

### Test case to add

A fixture pair `outer.proto` (imports inner) + `inner.proto`. Run via
CLI; assert generated output either compiles OR fails loudly with a
helpful "use protoc plugin" error.

---

## Bug 4 — Plugin mode produces colliding `class_name` when basenames repeat

### Symptom

The schema has files at multiple paths sharing the same basename:

```
uzir/common/v1/common.proto
uzir/assetpack/v1/common.proto
uzir/assetpack/client/v1/common.proto
uzir/assetpack/server/v1/common.proto

uzir/assetpack/client/v1/world.proto
uzir/assetpack/server/v1/world.proto
uzir/netcode/game/v1/world.proto
```

Plugin output:

```
uzir/common/v1/common.pb.gd            → class_name CommonProto
uzir/assetpack/v1/common.pb.gd         → class_name CommonProto   ← collision
uzir/assetpack/client/v1/common.pb.gd  → class_name CommonProto   ← collision
uzir/assetpack/server/v1/common.pb.gd  → class_name CommonProto   ← collision
uzir/assetpack/client/v1/world.pb.gd   → class_name WorldProto
uzir/assetpack/server/v1/world.pb.gd   → class_name WorldProto    ← collision
uzir/netcode/game/v1/world.pb.gd       → class_name WorldProto    ← collision
```

GDScript's `class_name` is **global across the entire project**.
Duplicates produce:

```
ERROR: Class "CommonProto" hides a global script class.
```

### Reproduction

Two `.proto` files in different packages but with the same basename
(e.g., `pkg_a/v1/util.proto`, `pkg_b/v1/util.proto`). Run protoc with
`--gdscript_out` over both. The generated `.pb.gd` files both declare
`class_name UtilProto`.

### Expected

Wrapper class names must be globally unique. Suggested strategy: derive
the class name from the package + basename, e.g.:

- `uzir/common/v1/common.proto` → `class_name UzirCommonV1CommonProto`
- `uzir/assetpack/v1/common.proto` → `class_name UzirAssetpackV1CommonProto`
- `uzir/netcode/game/v1/world.proto` → `class_name UzirNetcodeGameV1WorldProto`

(Or any deterministic scheme that incorporates the proto `package`
declaration. Long names are fine — GDScript identifiers don't care.)

### Why this matters

Plugin mode is the only mode that produces functional cross-file
references (Bug 3), so any project with multiple packages-with-shared-basenames
hits this immediately. It is the single largest blocker for using
gdproto-generated output in our project.

### Source pointer

`src/gdproto/plugin.py` contains a `_to_snake_case` helper visible in the
file header — the wrapper class name is presumably derived from a
similar function operating on basename only. The fix is to feed the
generator the proto package as part of the wrapper-name input.

### Test case to add

A fixture with two proto files in different packages but the same
basename:

```
fixtures/duplicates/pkg_a/util.proto    package pkg_a;
fixtures/duplicates/pkg_b/util.proto    package pkg_b;
```

Run plugin generation; assert the two generated `.pb.gd` files declare
**different** `class_name` values. Assert that loading both into a Godot
4.5 project (e.g., via `godot --headless --check-only`) does not error
on global class collisions.

---

## Bug 5 (minor) — Standalone CLI does not emit `proto_core_utils.gd`

### Symptom

Generated CLI output references `ProtoCoreUtils.encode_varint(...)`,
`ProtoCoreUtils.encode_string(...)`, etc., but no `proto_core_utils.gd`
runtime is emitted. Plugin mode emits it correctly (next to the per-file
output).

### Reproduction

```bash
gdproto demo.proto -o /tmp/out/demo.gd
ls /tmp/out/
# only demo.gd; no proto_core_utils.gd
```

### Expected

CLI mode should also emit `proto_core_utils.gd` to the output directory
(or the caller-specified `--runtime-dir`). Otherwise the generated code
is non-functional out of the box.

### Workaround

Manually copy `proto_core_utils.gd` from
`examples/buf/gen/gdscript/proto_core_utils.gd`.

### Why minor

Plugin mode handles it correctly. If Bug 3 is resolved by deprecating CLI
mode for multi-file projects, this becomes moot.

---

## Suggested resolution order

Bugs 1, 2, 4 are the load-bearing ones for our integration. With those
fixed:

1. We delete `protocol/godot-protocol/` (the service-strip fork).
2. We revert the canonical schema renames (`text` → `message`,
   `choice` → `option`).
3. We swap our `task gen-gdscript` from the standalone CLI to
   `protoc --gdscript_out=...` plugin invocation.
4. We delete the hand-written GDScript Envelope codec we are about to
   land as a stopgap.

Bug 3 (CLI cross-file) and Bug 5 (CLI runtime emission) become moot once
plugin mode is the documented path for projects with imports — which is
fine for us.

## Appendix — stopgap workarounds currently in our repo

For visibility while gdproto is being fixed, the Uzir monorepo has these
in-place workarounds (all are throwaway, removed when bugs above land):

- `protocol/godot-protocol/` — copies of `auth.proto` and `game.proto`
  with `service` blocks stripped (Bug 1).
- Canonical field renames: `ActionRejected.message → text` and the two
  `DialogueOption.option → choice` (Bug 2). Wire-compatible (field numbers
  preserved); `buf breaking` flags them and we currently bypass that gate.
- A hand-written GDScript Envelope codec (about to be added) for the
  three transport-layer messages (Hello / Welcome / Disconnect), bypassing
  gdproto generation entirely for the Foundation milestone.

Tracking issue on the Uzir side: cafecito-games/uzir#198 (Follow-up #5).

---

## Bug 6 — Enum fields encoded with wire type 2 instead of wire type 0

### Symptom

A protobuf message generated by gdproto-encoded GDScript fails to decode in
`protoc-gen-go`-generated Go code:

```
proto: cannot parse invalid wire-format data
```

Same message round-trips fine within GDScript (decoder uses varint and
encoder writes the matching value bytes).

### Reproduction

Hello message with `Platform.PLATFORM_DESKTOP` (=1):

```protobuf
syntax = "proto3";
package demo.v1;

enum Platform {
  PLATFORM_UNSPECIFIED = 0;
  PLATFORM_DESKTOP = 1;
}

message Hello {
  string access_token = 1;
  string client_version = 2;
  Platform platform = 3;
}
```

GDScript produces:
```
0a 03 74 6f 6b   12 05 30 2e 30 2e 31   1a 01
^^ field 1 wire 2  ^^ field 2 wire 2     ^^ field 3 wire 2  ← BUG
```

The enum tag is `0x1a` = `(3 << 3) | 2`. Correct would be
`0x18` = `(3 << 3) | 0` (varint). Patching that one byte makes the
message decode in protoc-generated decoders.

### Source pointer

`protocol/gen/gdscript/uzir/netcode/transport/v1/envelope.pb.gd` (the
generated file in our repo, but representative of any gdproto enum
encode):

```gdscript
# Field platform
if _platform != 0:
    result.append_array(ProtoCoreUtils.encode_varint(26))   # ← tag 26 = 0x1a
    result.append_array(ProtoCoreUtils.encode_varint(_platform))
```

The decoder side is correct (uses `decode_varint`), so internal GDScript
round-trips work. The encoder writes the wrong tag, so cross-implementation
decoding fails.

### Suggested fix

When generating the encode block for an enum field, the tag should be
`(field_number << 3) | 0` (varint), not `(field_number << 3) | 2`. This
likely lives in the same code path that generates int32/varint fields —
enums share their wire encoding with int32/varint primitive types, not
with messages or strings.

### Why this matters

Enums are pervasive (Platform, EntityType, SpawnCause, PlayerStatus,
PB_ERR, ...). Any generated GDScript message containing a non-default
enum value cannot be decoded by canonical protoc-generated decoders.

### Stopgap workaround in this repo

`game/client-godot/scripts/net/envelope_codec.gd::encode_hello` does NOT
call `set_platform()`, leaving the enum at default
`PLATFORM_UNSPECIFIED`. Proto3 omits default values, so the wrong-tagged
bytes are never emitted. Once Bug 6 is fixed, the codec should set
platform back to `PLATFORM_DESKTOP` / `PLATFORM_IOS` / `PLATFORM_ANDROID`
based on `OS.get_name()`.
