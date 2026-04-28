# Parity with Python `gdproto`

`gogdproto` aims for byte-identical GDScript output for shared fixtures, but we deliberately diverge where Python has bugs that break interop with canonical protoc-generated decoders.

## Intentional deviations (bugs we fix)

### Enum fields encode with wire type 0 (varint), not 2 (length-delimited)

Python's `gdproto` emits enum-typed fields with the wrong tag: `(field_number << 3) | 2` instead of `(field_number << 3) | 0`. Per the proto3 wire format spec, enum fields must use varint (wire type 0). The Python bug means `protoc-gen-go` and other canonical decoders reject any message with a non-default enum field:

```
proto: cannot parse invalid wire-format data
```

The Python decoder still works because it dispatches by field number and reads varints regardless of the declared wire-type bits, but cross-implementation decoding fails.

We emit the correct wire type. As a result, our output for `examples/example.proto` differs from Python's `examples/example.gd` by exactly one byte (the `PlayerStatus status = 4` field's tag in `Player.to_bytes`: 32 instead of 34). Map values whose type is an enum get the same fix (`mapValueWireType`).

Regression test: `TestEnumFieldWireTypeIsVarint` in `internal/generator/generator_test.go`.

_Reported by user 2026-04-28; fixed same day._

## Otherwise

For everything else, our output matches Python's regenerated `examples/example.gd` (1300 lines) and `examples/proto_core_utils.gd` (387 lines) byte-for-byte. When Python evolves, we re-sync.
