package validator

import (
	"github.com/cafecito-games/gogdproto/internal/ast"
)

// validateOneof validates each field within a oneof. Oneof fields share the
// parent message's field-number/name space, so the caller passes in the same
// fieldNumbers and fieldNames maps used for regular fields. A repeated field
// inside a oneof is reported as an error.
func (v *validator) validateOneof(
	oneof *ast.Oneof,
	message *ast.Message,
	fieldNumbers map[int]string,
	fieldNames map[string]bool,
	scope string,
) {
	for _, field := range oneof.Fields {
		v.validateField(field, message, fieldNumbers, fieldNames, scope)
		if field.Repeated {
			v.addError("Oneof field cannot be repeated", field.Line, field.Column)
		}
	}
}
