package boolexpr

import (
	"fmt"
)

// UnknownVariableError is returned when an unknown variable is encountered.
type UnknownVariableError struct {
	VariableName string
}

// NewUnknownVariableError creates a new UnknownVariableError with the given variable name.
func NewUnknownVariableError(variableName string) error {
	return &UnknownVariableError{VariableName: variableName}
}

func (e UnknownVariableError) Error() string {
	return fmt.Sprintf("unknown variable: %s", e.VariableName)
}
