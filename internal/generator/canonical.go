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
// filename is used only to place parse errors. An error means gdproto
// emitted GDScript that does not parse, which is a generator bug: callers
// surface it rather than writing the unformatted source.
func Canonicalize(filename, source string) (string, error) {
	file, err := gdparser.ParseFile(filename, []byte(source))
	if err != nil {
		return "", fmt.Errorf("gdproto emitted GDScript that does not parse (%s): %w", filename, err)
	}

	formatted := format.FileWithOptions(file, format.GodotStyle())

	// The formatter rewrites the source it was handed, so confirm its own
	// output still parses before letting it reach disk.
	if _, err := gdparser.ParseFile(filename, []byte(formatted)); err != nil {
		return "", fmt.Errorf("formatted output does not parse (%s): %w", filename, err)
	}

	if !strings.HasSuffix(formatted, "\n") {
		formatted += "\n"
	}
	return formatted, nil
}
