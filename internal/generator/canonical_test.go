package generator_test

import (
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/generator"
)

func TestCanonicalizeFormatsToGodotStyle(t *testing.T) {
	// One blank line between top-level functions; canonical form uses two.
	source := "extends RefCounted\n\nfunc a() -> void:\n\tpass\n\nfunc b() -> void:\n\tpass\n"

	got, err := generator.Canonicalize("sample.gd", source)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	if !strings.Contains(got, "pass\n\n\nfunc b()") {
		t.Errorf("want two blank lines between top-level functions, got:\n%s", got)
	}
}

func TestCanonicalizeIsIdempotent(t *testing.T) {
	source := "extends RefCounted\n\nfunc a() -> void:\n\tpass\n\nfunc b() -> void:\n\tpass\n"

	once, err := generator.Canonicalize("sample.gd", source)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	twice, err := generator.Canonicalize("sample.gd", once)
	if err != nil {
		t.Fatalf("canonicalize twice: %v", err)
	}

	if once != twice {
		t.Errorf("not idempotent:\n--- once ---\n%s\n--- twice ---\n%s", once, twice)
	}
}

func TestCanonicalizeEndsWithExactlyOneNewline(t *testing.T) {
	// Trailing blank lines are the interesting direction: the result must be
	// one newline whether the input had none or several.
	got, err := generator.Canonicalize("sample.gd", "extends RefCounted\n\n\n")
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	if !strings.HasSuffix(got, "\n") || strings.HasSuffix(got, "\n\n") {
		t.Errorf("want exactly one trailing newline, got %q", got)
	}
}

func TestCanonicalizeRejectsSourceThatDoesNotParse(t *testing.T) {
	_, err := generator.Canonicalize("broken.gd", "func (((\n")
	if err == nil {
		t.Fatal("want an error for source that does not parse, got nil")
	}
	if !strings.Contains(err.Error(), "broken.gd") {
		t.Errorf("want the error to name the file, got %q", err.Error())
	}
}

func TestProtoCoreUtilsAssetIsAlreadyCanonical(t *testing.T) {
	source := generator.GenerateProtoCoreUtilsRaw()

	got, err := generator.Canonicalize("proto_core_utils.gd", source)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	if got != source {
		t.Error("proto_core_utils_data.gd is not in canonical format; " +
			"format it with gdkit and commit the result")
	}
}
