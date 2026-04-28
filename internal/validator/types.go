package validator

import (
	"github.com/cafecito-games/gogdproto/internal/ast"
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
