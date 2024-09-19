package boolexpr

import (
	"fmt"
	"strings"
)

// This is the entry point of the lexing
func buildTree(expression string) (*Node, error) {
	parts := splitString(expression)
	if len(parts) == 1 {
		return buildUnary(expression)
	}
	return buildBinary(parts)
}

func splitString(expression string) []string {
	var parts []string
	var currentPart string
	var parenthesesCount int

	for i := 0; i < len(expression); i++ {
		switch expression[i] {
		case ' ':
			if parenthesesCount == 0 {
				if currentPart != "" {
					parts = append(parts, removeWrappingParentheses(currentPart))
					currentPart = ""

					if len(parts) == 2 {
						rest := expression[i+1:]
						return append(parts, removeWrappingParentheses(rest))
					}
				}
			} else {
				currentPart += string(expression[i])
			}
		case '(':
			parenthesesCount++
			currentPart += string(expression[i])
		case ')':
			parenthesesCount--
			currentPart += string(expression[i])
		default:
			currentPart += string(expression[i])
		}
	}

	if currentPart != "" {
		parts = append(parts, removeWrappingParentheses(currentPart))
	}

	return parts
}

func removeWrappingParentheses(expression string) string {
	if len(expression) > 0 && expression[0] == '(' && expression[len(expression)-1] == ')' {
		return expression[1 : len(expression)-1]
	}
	return expression
}

func buildUnary(expression string) (*Node, error) {
	if strings.HasPrefix(expression, "!") {
		return parseNegation(expression[1:])
	}
	return parseLiteral(expression)
}

func buildBinary(parts []string) (*Node, error) {
	switch parts[1] {
	case "&&":
		return parseConjunction(parts[0], parts[2])
	case "||":
		return parseDisjunction(parts[0], parts[2])
	default:
		return nil, fmt.Errorf("invalid operator '%s'", parts[1])
	}
}

func parseLiteral(expression string) (*Node, error) {
	// Base case: if the expression is a single boolean value
	/*value, err := strconv.ParseBool(expression)
	if err != nil {
		// Wrap the error and return it
		return nil, fmt.Errorf("failed to parse boolean value '%s': %w", expression, err)
	}*/
	// TODO: Remove value??
	return &Node{
		Operator:        LITERAL,
		rawLiteralValue: expression,
		//Value:    &value,
	}, nil
}

func parseNegation(expression string) (*Node, error) {
	left, err := buildTree(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to build left subtree: %w", err)
	}
	return &Node{
		Operator: NOT,
		Left:     left,
	}, nil
}

func parseConjunction(leftExpression, rightExpression string) (*Node, error) {
	left, err := buildTree(leftExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to build left subtree: %w", err)
	}
	right, err := buildTree(rightExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to build right subtree: %w", err)
	}
	return &Node{
		Operator: AND,
		Left:     left,
		Right:    right,
	}, nil
}

func parseDisjunction(leftExpression, rightExpression string) (*Node, error) {
	left, err := buildTree(leftExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to build left subtree: %w", err)
	}
	right, err := buildTree(rightExpression)
	if err != nil {
		return nil, fmt.Errorf("failed to build right subtree: %w", err)
	}
	return &Node{
		Operator: OR,
		Left:     left,
		Right:    right,
	}, nil
}
