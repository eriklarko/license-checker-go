package boolexpr

import (
	"fmt"
)

type Operator int

const (
	LITERAL Operator = iota
	NOT
	AND
	OR
)

type Node struct {
	Operator Operator
	Left     *Node
	Right    *Node

	rawLiteralValue string
}

// New creates a new solvable boolean expression based on the given input string
// Example usage:
//
//	tree, err := boolexpr.New("T && (T || F)")
//	if err != nil {
//		fmt.Fatalf("failed to create decision tree: %v", err)
//	}
//	fmt.Println(tree.Solve()) // Output: true
func New(expression string) (*Node, error) {
	root, err := buildTree(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to build decision tree for expression '%s': %w", expression, err)
	}
	return root, nil
}
