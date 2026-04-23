package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExpressionsInCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Command
	}{
		{
			name:  "simple expression in command",
			input: `task test { echo {{target}} }`,
			expected: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "echo "},
					ExpressionElement{Expression: Identifier{Name: "target"}},
				}},
			},
		},
		{
			name:  "access expression in command",
			input: `task deploy { echo "API: {{env.API_KEY}}" }`,
			expected: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "echo \"API: "},
					ExpressionElement{Expression: AccessId{
						Object:   Identifier{Name: "env"},
						Property: "API_KEY",
					}},
					StringElement{Value: "\""},
				}},
			},
		},
		{
			name:  "or expression in command",
			input: `task build { make {{target || "release"}} }`,
			expected: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "make "},
					ExpressionElement{Expression: Or{
						Left:  Identifier{Name: "target"},
						Right: StringLiteral{Value: "release"},
					}},
				}},
			},
		},
		{
			name:  "complex expression in command",
			input: `task deploy { deploy --env={{env.DEPLOY_ENV || "development"}} }`,
			expected: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "deploy --env="},
					ExpressionElement{Expression: Or{
						Left:  AccessId{Object: Identifier{Name: "env"}, Property: "DEPLOY_ENV"},
						Right: StringLiteral{Value: "development"},
					}},
				}},
			},
		},
		{
			name:  "multiple expressions in command",
			input: `task info { echo "{{app}} v{{version}}" }`,
			expected: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "echo \""},
					ExpressionElement{Expression: Identifier{Name: "app"}},
					StringElement{Value: " v"},
					ExpressionElement{Expression: Identifier{Name: "version"}},
					StringElement{Value: "\""},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok, err := parseNoPos(tt.input)
			require.True(t, ok, "parsing should succeed")
			require.NoError(t, err, "should not return error")

			require.Len(t, result.Tasks, 1, "should have one task")
			require.Equal(t, tt.expected, result.Tasks[0].Commands)
		})
	}
}

func TestParseExpressionsWithSpacing(t *testing.T) {
	// Test that expressions handle spacing inside {{}}
	input := `task test { echo {{ env.API_KEY || "default" }} }`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := []Command{
		{Elements: []CommandElement{
			StringElement{Value: "echo "},
			ExpressionElement{Expression: Or{
				Left:  AccessId{Object: Identifier{Name: "env"}, Property: "API_KEY"},
				Right: StringLiteral{Value: "default"},
			}},
		}},
	}

	require.Len(t, result.Tasks, 1, "should have one task")
	require.Equal(t, expected, result.Tasks[0].Commands)
}
