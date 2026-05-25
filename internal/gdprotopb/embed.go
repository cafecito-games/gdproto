// Package gdprotopb contains the generated Go bindings for
// proto/gdproto/options.proto, along with an embedded copy of the proto
// source so callers can hand it to users (e.g. via --print-options-proto).
package gdprotopb

import _ "embed"

//go:embed options.proto
var optionsProto []byte

// Bytes returns the embedded gdproto/options.proto source.
// The slice is a fresh copy; callers may modify it freely.
func Bytes() []byte {
	out := make([]byte, len(optionsProto))
	copy(out, optionsProto)
	return out
}
