package generator

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

// uidPrefix precedes every Godot resource identifier.
const uidPrefix = "uid://"

// uidAlphabet holds the characters Godot renders an identifier with, in digit
// order, and so fixes the base at 34. It omits z and 9 deliberately: Godot
// computes its digit count as 'z' - 'a', which is 25 rather than the 26
// letters, and its base as that plus '9' - '0', which is 34 rather than 36.
// Godot's source marks the off-by-one as unfixable, because correcting it
// would change the meaning of every identifier already written to disk.
const uidAlphabet = "abcdefghijklmnopqrstuvwxy012345678"

// maxUID is the largest identifier Godot can name: it masks the bits it draws
// so the sign bit of its signed identifier type is never set.
const maxUID uint64 = 0x7FFFFFFFFFFFFFFF

// DeriveUID returns the uid:// identifier for a generated file, derived from
// its name so that regenerating a schema reproduces the same identifier.
//
// Godot draws these at random, but a generator runs on every schema sync, and
// reissuing an identifier changes what every existing uid:// reference to that
// file resolves to. Derivation keeps regeneration byte-identical and makes two
// checkouts agree.
func DeriveUID(filename string) string {
	digest := sha256.Sum256([]byte(filename))
	return encodeUID(binary.BigEndian.Uint64(digest[:8]) & maxUID)
}

// encodeUID renders an identifier the way Godot's ResourceUID::id_to_text
// does: base 34 over uidAlphabet, most significant digit first, behind
// uidPrefix.
func encodeUID(id uint64) string {
	// Dividing yields the least significant digit first, so digits are
	// collected backwards and reversed into the text.
	digits := make([]byte, 0, 13)
	for {
		digits = append(digits, uidAlphabet[id%uint64(len(uidAlphabet))])
		id /= uint64(len(uidAlphabet))
		if id == 0 {
			break
		}
	}

	var text strings.Builder
	text.Grow(len(uidPrefix) + len(digits))
	text.WriteString(uidPrefix)
	for i := len(digits) - 1; i >= 0; i-- {
		text.WriteByte(digits[i])
	}
	return text.String()
}

// SidecarFilename returns the .uid sidecar path for a generated file.
func SidecarFilename(filename string) string { return filename + ".uid" }

// SidecarSource returns the contents of the .uid sidecar for a generated file.
func SidecarSource(filename string) string { return DeriveUID(filename) + "\n" }
