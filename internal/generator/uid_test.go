package generator_test

import (
	"strings"
	"testing"

	gdkituid "github.com/cafecito-games/gdkit/uid"

	"github.com/cafecito-games/gdproto/internal/generator"
)

func TestDeriveUIDIsStableForTheSameFilename(t *testing.T) {
	first := generator.DeriveUID("ExamplePlayer.pb.gd")
	second := generator.DeriveUID("ExamplePlayer.pb.gd")

	if first != second {
		t.Errorf("derivation is not stable: %q then %q", first, second)
	}
	if !strings.HasPrefix(first, "uid://") {
		t.Errorf("want a uid:// identifier, got %q", first)
	}
}

func TestDeriveUIDDiffersBetweenFilenames(t *testing.T) {
	if a, b := generator.DeriveUID("A.pb.gd"), generator.DeriveUID("B.pb.gd"); a == b {
		t.Errorf("distinct filenames share an identifier: %q", a)
	}
}

func TestDerivedUIDDecodesWithGdkit(t *testing.T) {
	for _, filename := range []string{
		"ExamplePlayer.pb.gd",
		"ExampleGameState.pb.gd",
		"proto_core_utils.gd",
	} {
		text := generator.DeriveUID(filename)
		id, valid := gdkituid.Decode(text)
		if !valid {
			t.Errorf("%s: gdkit rejects %q", filename, text)
			continue
		}
		if id > gdkituid.MaxID {
			t.Errorf("%s: id %d exceeds MaxID", filename, id)
		}
		// Round-tripping through gdkit's encoder proves the two agree on the
		// alphabet, the base and the digit order, not merely on decodability.
		if again := gdkituid.Encode(id); again != text {
			t.Errorf("%s: gdkit encodes %d as %q, gdproto as %q", filename, id, again, text)
		}
	}
}
