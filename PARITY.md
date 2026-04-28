# Parity with Python `gdproto`

`gogdproto` aims for byte-identical GDScript output for shared fixtures. As of
M7 (2026-04-28), our output for `examples/example.proto` is byte-identical to
Python's regenerated `examples/example.gd` (1300 lines + 387-line sibling
`proto_core_utils.gd`).

No outstanding deviations. When Python evolves, we re-sync.
