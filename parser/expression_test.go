package parser

import (
	"testing"

	p "github.com/lab47/peggysue"
	"github.com/stretchr/testify/require"
)

func TestParseExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Expression
	}{
		{
			name:     "simple identifier",
			input:    "target",
			expected: Identifier{Name: "target"},
		},
		{
			name:     "access expression",
			input:    "env.API_KEY",
			expected: AccessId{Object: Identifier{Name: "env"}, Property: "API_KEY"},
		},
		{
			name:     "string literal",
			input:    `"development"`,
			expected: StringLiteral{Value: "development"},
		},
		{
			name:  "or expression",
			input: `target || "release"`,
			expected: Or{
				Left:  Identifier{Name: "target"},
				Right: StringLiteral{Value: "release"},
			},
		},
		{
			name:  "complex expression",
			input: `env.DEPLOY_ENV || "development"`,
			expected: Or{
				Left:  AccessId{Object: Identifier{Name: "env"}, Property: "DEPLOY_ENV"},
				Right: StringLiteral{Value: "development"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := p.New()
			grammar := NewGrammar()

			result, ok, err := parser.Parse(grammar.expr, tt.input)
			require.True(t, ok, "parsing should succeed")
			require.NoError(t, err, "should not return error")

			// Actions return pointer Expressions so peggysue can set Position.
			// Unwrap to the value form and strip positions for the comparison.
			got := zeroExprPosition(exprValue(result.(Expression)))
			require.Equal(t, tt.expected, got)
		})
	}
}
