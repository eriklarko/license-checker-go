package boolexpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitString(t *testing.T) {
	testCases := map[string][]string{
		"a":      {"a"},
		"a && b": {"a", "&&", "b"},
		"a || b": {"a", "||", "b"},

		"(a)":      {"a"},
		"(a && b)": {"a && b"},
		"(a) && b": {"a", "&&", "b"},

		"(a && b) && c": {"a && b", "&&", "c"},
		"(a || b) && c": {"a || b", "&&", "c"},
		"a && (b && c)": {"a", "&&", "b && c"},
		"a && (b || c)": {"a", "&&", "b || c"},

		"a && b && c": {"a", "&&", "b && c"},

		"a && (b && (c && d))": {"a", "&&", "b && (c && d)"},
	}

	for expression, expected := range testCases {
		t.Run(expression, func(t *testing.T) {
			result := splitString(expression)
			assert.Equal(t, expected, result)
		})
	}
}
func TestBuildTree(t *testing.T) {
	testCases := map[string]*Node{
		"t": {
			Operator:        LITERAL,
			rawLiteralValue: "t",
		},
		"f": {
			Operator:        LITERAL,
			rawLiteralValue: "f",
		},

		"SomeVariable": {
			Operator:        LITERAL,
			rawLiteralValue: "SomeVariable",
			// the value of variables are not known yet. depends on the `context`
			// passed to `Solve`
		},

		"!t": {
			Operator: NOT,
			Left: &Node{
				Operator:        LITERAL,
				rawLiteralValue: "t",
			},
		},

		"t && f": {
			Operator: AND,
			Left: &Node{
				Operator:        LITERAL,
				rawLiteralValue: "t",
			},
			Right: &Node{
				Operator:        LITERAL,
				rawLiteralValue: "f",
			},
		},
		"t || f": {
			Operator: OR,
			Left: &Node{
				Operator:        LITERAL,
				rawLiteralValue: "t",
			},
			Right: &Node{
				Operator:        LITERAL,
				rawLiteralValue: "f",
			},
		},
		"t && (f || t)": {
			Operator: AND,
			Left: &Node{
				Operator:        LITERAL,
				rawLiteralValue: "t",
			},
			Right: &Node{
				Operator: OR,
				Left: &Node{
					Operator:        LITERAL,
					rawLiteralValue: "f",
				},
				Right: &Node{
					Operator:        LITERAL,
					rawLiteralValue: "t",
				},
			},
		},
	}

	for expression, expected := range testCases {
		t.Run(expression, func(t *testing.T) {
			result, err := buildTree(expression)
			require.NoError(t, err)

			assert.Equal(t, expected, result)
		})
	}
}
