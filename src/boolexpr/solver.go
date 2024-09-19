package boolexpr

import (
	"fmt"
	"strconv"
)

func (n *Node) Solve(context map[string]bool) (bool, error) {
	if n.Operator == LITERAL {
		v, err := n.value(context)
		if v == nil {
			return false, fmt.Errorf("no value found for literal '%s'", n.rawLiteralValue)
		}
		return *v, err
	}

	if n.Operator == NOT {
		result, err := n.Left.Solve(context)
		if err != nil {
			return false, fmt.Errorf("failed solving NOT sub-expression: %w", err)
		}
		return !result, nil
	}

	leftResult, err := n.Left.Solve(context)
	if err != nil {
		return false, fmt.Errorf("failed solving left expression: %w", err)
	}
	rightResult, err := n.Right.Solve(context)
	if err != nil {
		return false, fmt.Errorf("failed solving right expression: %w", err)
	}

	if n.Operator == AND {
		return leftResult && rightResult, nil
	}
	if n.Operator == OR {
		return leftResult || rightResult, nil
	}

	return false, fmt.Errorf("unknown operator: %v", n.Operator)
}

func (n *Node) value(context map[string]bool) (*bool, error) {
	if n.rawLiteralValue == "" {
		return nil, nil
	}

	// is the literal value a variable?
	if val, ok := context[n.rawLiteralValue]; ok {
		return &val, nil
	}

	// is the literal value a boolean?
	value, err := strconv.ParseBool(n.rawLiteralValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse boolean value '%s': %w", n.rawLiteralValue, err)
	}
	return &value, nil
}
