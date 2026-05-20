package validator

import (
	"fmt"
	"strings"

	"github.com/cafecito-games/gdproto/internal/ast"
)

// buildTypeRegistry populates v.definedTypes with every message and enum
// name visible in the file. Top-level types are registered by their bare
// name; nested types are registered as "<prefix>.<name>". When message is
// nil, the call walks the file's top-level declarations and recurses into
// each message; otherwise it walks the supplied message's nested types.
func (v *validator) buildTypeRegistry(message *ast.Message, prefix string) {
	if message == nil {
		for _, enum := range v.file.Enums {
			v.definedTypes[enum.Name] = true
		}
		for _, msg := range v.file.Messages {
			v.definedTypes[msg.Name] = true
			v.buildTypeRegistry(msg, msg.Name)
		}
		return
	}

	for _, enum := range message.NestedEnums {
		v.definedTypes[prefix+"."+enum.Name] = true
	}
	for _, nested := range message.NestedMessages {
		nestedPrefix := prefix + "." + nested.Name
		v.definedTypes[nestedPrefix] = true
		v.buildTypeRegistry(nested, nestedPrefix)
	}
}

// validateFieldType resolves a type reference against the registered type
// set. Scalar types and types whose source file has been set by the importer
// are accepted unconditionally. Other references are normalized (leading "."
// stripped), looked up directly, then resolved through the file's package
// prefix and finally walked up the enclosing scope to permit nested-type
// shorthand. Unresolved references produce an "Undefined type" error.
func (v *validator) validateFieldType(fieldType string, line, column int, currentScope, fullTypePath, sourceFile string) {
	if scalarTypes[fieldType] {
		return
	}

	if sourceFile != "" {
		return
	}

	normalizedType := strings.TrimLeft(fieldType, ".")

	if fullTypePath != "" && v.definedTypes[fullTypePath] {
		return
	}

	if v.definedTypes[normalizedType] {
		return
	}

	if v.file.Package != "" {
		packagePrefix := v.file.Package + "."
		if strings.HasPrefix(normalizedType, packagePrefix) {
			packageRelative := normalizedType[len(packagePrefix):]
			if v.definedTypes[packageRelative] {
				return
			}
		}
	}

	if currentScope != "" {
		typeParts := strings.Split(normalizedType, ".")
		scopeParts := strings.Split(currentScope, ".")
		for index := len(scopeParts); index > 0; index-- {
			candidateParts := append([]string{}, scopeParts[:index]...)
			candidateParts = append(candidateParts, typeParts...)
			candidate := strings.Join(candidateParts, ".")
			if v.definedTypes[candidate] {
				return
			}
		}
	}

	v.addError(fmt.Sprintf("Undefined type %q", fieldType), line, column)
}
