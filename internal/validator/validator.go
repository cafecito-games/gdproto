package validator

import (
	"fmt"

	"github.com/cafecito-games/gogdproto/internal/ast"
)

// validator carries state across a single validation pass over a ProtoFile.
type validator struct {
	file         *ast.ProtoFile
	filename     string
	errors       []ValidationError
	definedTypes map[string]bool
}

// Validate performs semantic validation on file and returns the resulting
// errors in source order. A nil slice is returned when no errors are found.
func Validate(file *ast.ProtoFile, filename string) []ValidationError {
	v := &validator{
		file:         file,
		filename:     filename,
		definedTypes: make(map[string]bool),
	}
	v.validate()
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

// addError records a validation error at the given source position.
func (v *validator) addError(message string, line, column int) {
	v.errors = append(v.errors, ValidationError{
		File:    v.filename,
		Line:    line,
		Column:  column,
		Message: message,
	})
}

// validate orchestrates the full validation pipeline.
func (v *validator) validate() {
	v.validateSyntax()
	v.buildTypeRegistry(nil, "")
	// Enum and message validation will be wired in subsequent tasks.
}

// validateSyntax checks that the file declares proto3.
func (v *validator) validateSyntax() {
	if v.file.Syntax != "proto3" {
		v.addError(
			fmt.Sprintf("Unsupported syntax version: %q (only proto3 is supported)", v.file.Syntax),
			v.file.Line,
			v.file.Column,
		)
	}
}
