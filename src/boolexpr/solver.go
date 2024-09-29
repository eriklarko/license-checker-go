package boolexpr

import (
	"fmt"
	"strconv"
)

func (n *Node) Solve(context map[string]bool) (bool, error) {
	if n.Operator == LITERAL {
		v, err := n.value(context)
		if err != nil {
			return false, err
		}
		if v == nil {
			return false, NewUnknownVariableError(n.rawLiteralValue)
		}

		return *v, nil
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
	// is the literal value a boolean?
	value, err := strconv.ParseBool(n.rawLiteralValue)
	if err == nil {
		return &value, nil
		// if the error is not nil, it's likely that we're trying to get the
		// value of a variable, so we'll continue to the next check and ignore
		// the error
	}

	// is the literal value a known variable?
	if val, ok := context[n.rawLiteralValue]; ok {
		return &val, nil
	}

	// the variable is unknown, which is not an error
	return nil, nil
}
