package gdast

import "strings"

// GDScript is the root container for a GDScript file. Items are joined with
// newlines when rendered.
type GDScript struct {
	Items []Node
}

// ToGDScript renders each item on its own line at the given indent level.
func (s GDScript) ToGDScript(indentLevel int) string {
	var sb strings.Builder
	for i, item := range s.Items {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(item.ToGDScript(indentLevel))
	}
	return sb.String()
}
