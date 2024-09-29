package boolexpr_test

import (
	"testing"

	"github.com/eriklarko/license-checker/src/boolexpr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiterals(t *testing.T) {
	tests := map[string]bool{
		"T": true,
		"F": false,
	}
	runSolverTests(t, tests, make(map[string]bool))
}

func TestVariables(t *testing.T) {
	context := map[string]bool{
		"A": true,
		"B": false,
	}
	tests := map[string]bool{
		"A": true,  // A is true in the context
		"B": false, // B is false in the context

		"!A": false,
		"!B": true,

		"A && B": false,
		"A || B": true,
	}
	runSolverTests(t, tests, context)
}

func TestNot(t *testing.T) {
	tests := map[string]bool{
		"!T": false,
		"!F": true,
	}
	runSolverTests(t, tests, make(map[string]bool))
}

func TestAnd(t *testing.T) {
	tests := map[string]bool{
		"T && T": true,
		"T && F": false,
		"F && T": false,
		"F && F": false,
	}
	runSolverTests(t, tests, make(map[string]bool))
}

func TestOr(t *testing.T) {
	tests := map[string]bool{
		"T || T": true,
		"T || F": true,
		"F || T": true,
		"F || F": false,
	}
	runSolverTests(t, tests, make(map[string]bool))
}

func TestRecursiveExpressions(t *testing.T) {
	tests := map[string]bool{
		"T && !F": true,
		"!F && T": true,

		"T || (F && F)": true,
		"(T || F) && F": false,

		"T && T && T": true,
		"T && T && F": false,
	}
	runSolverTests(t, tests, make(map[string]bool))
}

func runSolverTests(t *testing.T, tests map[string]bool, context map[string]bool) {
	for expression, expected := range tests {
		t.Run(expression, func(t *testing.T) {
			node, err := boolexpr.New(expression)
			require.NoError(t, err)

			result, err := node.Solve(context)
			require.NoError(t, err)
			assert.Equal(t, expected, result)
		})
	}
}

func TestUnknownVariable(t *testing.T) {
	// create an expression referencing variable A
	node, err := boolexpr.New("A")
	require.NoError(t, err)

	// and try to solve it without providing a value for A
	_, err = node.Solve(make(map[string]bool))
	assert.Contains(t, err.Error(), "unknown variable")
	assert.Contains(t, err.Error(), "A")
}
