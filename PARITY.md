# Parity with Python `gdproto`

`gogdproto` aims for byte-identical GDScript output for shared fixtures. This file tracks intentional and unintentional deviations.

## Known Python bugs we faithfully reproduce

### Oneof fields missing from `from_bytes`

The Python generator emits oneof field cases in `to_bytes` but not in `from_bytes`. As a result, oneof-encoded data round-trips incorrectly: the `to_bytes` output is correct, but `from_bytes` silently drops oneof field data because there's no matching `case` in the field-number `match`.

We reproduce this bug to maintain byte-identical output with Python's golden. When the Python tool fixes this, regenerate the golden and update our generator to iterate `m.Oneofs` in `generateFromBytes`. The per-field decoder already supports oneof tracking via `OneofParent`.

_Reported by M5-T4 implementer (2026-04-28)._
