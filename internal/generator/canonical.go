package generator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gdparser"
	"github.com/cafecito-games/gdparser/format"
)

// Canonicalize returns source in the canonical GDScript form that gdkit's
// formatter produces. gdkit's `format` tool delegates to gdparser's
// formatter, and gdkit's default config is field-for-field gdparser's
// GodotStyle, so output that has been through this function passes
// `gdkit format check` against gdkit's defaults.
//
// filename is used to place errors. An error means gdproto emitted GDScript
// that does not parse, or that the formatter is not idempotent over it —
// either way a bug, so callers surface it rather than writing the source.
func Canonicalize(filename, source string) (string, error) {
	file, err := gdparser.ParseFile(filename, []byte(source))
	if err != nil {
		return "", fmt.Errorf("gdproto emitted GDScript that does not parse: %w", err)
	}

	formatted := format.FileWithOptions(file, format.GodotStyle())

	reparsed, err := gdparser.ParseFile(filename, []byte(formatted))
	if err != nil {
		return "", fmt.Errorf("formatted output does not parse: %w", err)
	}

	// `gdkit format check` reports a file when a formatting pass would change
	// it, so conformance rests on the formatter being idempotent rather than
	// merely producing parseable output. Checking that here means a gdparser
	// regression surfaces as a generation failure instead of as output that
	// looks canonical and fails in every downstream project.
	if again := format.FileWithOptions(reparsed, format.GodotStyle()); again != formatted {
		return "", fmt.Errorf("formatting %s is not idempotent: a second pass changes the output", filename)
	}

	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}
	return formatted, nil
}
