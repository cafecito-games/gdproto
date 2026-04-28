# Parity with Python `gdproto`

`gogdproto` aims for byte-identical GDScript output for shared fixtures. This file tracks intentional deviations from the Python reference.

## Intentional deviations (bugs we fix)

### Oneof fields decoded by `from_bytes`

The Python generator emits oneof field cases in `to_bytes` but not in `from_bytes`, which means oneof-encoded data silently drops on the round trip. We emit the cases. As a result, our golden (`examples/golden.gd`) diverges from the Python repo's `examples/example.gd` by ~22 lines covering the missing oneof cases.

_Fixed 2026-04-28._

## Notes on Python's evolution

Python `gdproto`'s `examples/example.gd` is stale relative to its own generator: regenerating from the current Python tool produces a substantially different file (~1300 lines, with `class_name` directive, sibling `proto_core_utils.gd` runtime file, etc.). We deliberately track the older 746-line format because it's what the Python repo ships as its committed example, and chasing the moving target of Python's current output isn't worth the churn.

If/when Python's repo regenerates `examples/example.gd`, we should re-evaluate.
