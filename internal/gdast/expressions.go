package gdast

import "strings"

// RawExpression embeds an opaque snippet of GDScript source in expression
// position. Multi-line snippets have non-blank lines prefixed with the
// requested indentation; blank lines remain empty.
type RawExpression struct {
	Code string
}

// ToGDScript prefixes every non-blank line of Code with the requested
// indentation, preserving relative tabs already present in the snippet.
func (r RawExpression) ToGDScript(level int) string {
	if level == 0 {
		return r.Code
	}
	if !strings.Contains(r.Code, "\n") {
		return indent(level) + r.Code
	}
	lines := strings.Split(r.Code, "\n")
	pad := indent(level)
	out := make([]string, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			out[i] = line
		} else {
			out[i] = pad + line
		}
	}
	return strings.Join(out, "\n")
}

func (RawExpression) expression() {}
