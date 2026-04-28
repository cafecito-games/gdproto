package generator

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/cafecito-games/gogdproto/internal/ast"
	"github.com/cafecito-games/gogdproto/internal/gdast"
)

// Generate produces a gdast ClassDefinition representing the GDScript
// translation of the proto file. The sourceName is the input filename used in
// the file header comment.
func Generate(file *ast.ProtoFile, sourceName string) (*gdast.ClassDefinition, error) {
	g := &generator{file: file, sourceName: sourceName, enumTypes: map[string]bool{}}
	g.collectEnumTypes()
	return g.generate()
}

type generator struct {
	file       *ast.ProtoFile
	sourceName string
	enumTypes  map[string]bool
}

func (g *generator) collectEnumTypes() {
	for _, e := range g.file.Enums {
		g.enumTypes[e.Name] = true
	}
	for _, m := range g.file.Messages {
		g.collectMessageEnumTypes(m)
	}
}

func (g *generator) collectMessageEnumTypes(m *ast.Message) {
	for _, e := range m.NestedEnums {
		g.enumTypes[e.Name] = true
	}
	for _, nested := range m.NestedMessages {
		g.collectMessageEnumTypes(nested)
	}
}

func (g *generator) generate() (*gdast.ClassDefinition, error) {
	var statements []gdast.Node
	appendItem := func(node gdast.Node) {
		if len(statements) > 0 {
			statements = append(statements, gdast.EmptyLine{}, gdast.EmptyLine{}, gdast.EmptyLine{})
		}
		statements = append(statements, node)
	}

	for _, e := range g.file.Enums {
		appendItem(generateEnum(e))
	}
	for _, m := range g.file.Messages {
		appendItem(g.generateMessage(m))
	}

	return &gdast.ClassDefinition{
		ClassNameDirective: wrapperClassName(g.sourceName),
		Extends:            "RefCounted",
		HeaderComment:      headerCommentText(headerSourceName(g.sourceName)),
		Statements:         statements,
		TightStatements:    true,
	}, nil
}

// headerSourceName returns the basename of the input path used in the file
// header comment (e.g. "/tmp/foo/example.proto" -> "example.proto").
func headerSourceName(filename string) string {
	return filepath.Base(filename)
}

// nonAlphaNumericRun matches one or more characters that are not ASCII
// letters or digits. It is used by normalizeProtoStem to coerce arbitrary
// punctuation in a proto path component into single underscores.
var nonAlphaNumericRun = regexp.MustCompile(`[^A-Za-z0-9]+`)

// underscoreRun matches one or more consecutive underscores. After the snake-
// case conversion runs we collapse any runs created by it back to a single
// underscore so the final identifier is stable.
var underscoreRun = regexp.MustCompile(`_+`)

// wrapperClassName derives the GDScript `class_name` directive from the input
// proto path. It mirrors `_get_wrapper_class_name` from the upstream Python
// implementation: each path component is sanitized and snake-cased, then each
// piece is PascalCased, joined, and finally suffixed with "Proto".
func wrapperClassName(protoFile string) string {
	protoFile = strings.TrimSuffix(protoFile, ".proto")
	parts := strings.Split(protoFile, "/")
	var pieces []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		normalized := normalizeProtoStem(part)
		for _, sub := range strings.Split(normalized, "_") {
			if sub == "" {
				continue
			}
			pieces = append(pieces, capitalizeASCII(sub))
		}
	}
	return strings.Join(pieces, "") + "Proto"
}

// normalizeProtoStem converts an arbitrary path component into a stable
// snake_case identifier suitable for further PascalCase joining. The pipeline
// matches the Python helper of the same name: punctuation collapses to `_`,
// the result is snake-cased, repeated underscores collapse, and an empty or
// digit-leading result is rewritten to a safe identifier.
func normalizeProtoStem(name string) string {
	sanitized := strings.Trim(nonAlphaNumericRun.ReplaceAllString(name, "_"), "_")
	var snake string
	if sanitized != "" {
		snake = toSnakeCase(sanitized)
	} else {
		snake = "proto"
	}
	snake = strings.Trim(underscoreRun.ReplaceAllString(snake, "_"), "_")
	if snake == "" {
		snake = "proto"
	}
	if unicode.IsDigit(rune(snake[0])) {
		snake = "proto_" + snake
	}
	return snake
}

// toSnakeCase converts CamelCase or mixedCase input to snake_case. Runs of
// upper-case letters followed by a lower-case letter are split before the
// final upper-case letter ("HTTPServer" -> "http_server").
func toSnakeCase(name string) string {
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				next := rune(0)
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					b.WriteByte('_')
				} else if unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// capitalizeASCII returns the input with its first ASCII letter upper-cased
// and the remainder lower-cased, matching Python's `str.capitalize` behaviour
// for the identifier subset produced by normalizeProtoStem.
func capitalizeASCII(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}
	return string(runes)
}
