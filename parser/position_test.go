package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPosition_TaskTracksStartEndLine(t *testing.T) {
	input := "task hello {\n    echo \"Hello\"\n}\n"

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)

	pos := result.Tasks[0].Position
	require.Equal(t, 1, pos.Line, "task hello starts on line 1")
	require.Equal(t, 0, pos.Start, "task hello starts at byte 0")
	require.Greater(t, pos.End, pos.Start, "end should be after start")
	require.LessOrEqual(t, pos.End, len(input), "end must be within input")
}

func TestPosition_TaskOnLaterLineHasCorrectLine(t *testing.T) {
	input := "# top comment\n\ntask build {\n    go build\n}\n"

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)

	require.Equal(t, 3, result.Tasks[0].Position.Line, "task build starts on line 3")
}

func TestPosition_NamespaceAndNestedTask(t *testing.T) {
	input := "namespace db {\n    task migrate {\n        echo \"migrating\"\n    }\n}\n"

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Namespaces, 1)
	require.Len(t, result.Namespaces[0].Tasks, 1)

	ns := result.Namespaces[0]
	require.Equal(t, 1, ns.Position.Line, "namespace starts on line 1")

	migrate := ns.Tasks[0]
	require.Equal(t, 2, migrate.Position.Line, "nested task starts on line 2")
}

func TestPosition_VariableTracked(t *testing.T) {
	input := "VERSION = \"1.0.0\"\n"

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Variables, 1)

	v := result.Variables[0]
	require.Equal(t, 1, v.Position.Line)
	require.Equal(t, 0, v.Position.Start)
}

func TestPosition_FilenamePropagated(t *testing.T) {
	input := "task hello {\n    echo \"x\"\n}\n"

	result, ok, err := ParseQuakefileWithSource(input, "/tmp/Quakefile")
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)

	require.Equal(t, "/tmp/Quakefile", result.Tasks[0].Position.Filename)
}

func TestPosition_CommandElementPositionsZeroed(t *testing.T) {
	// parseCommands re-parses each command line in isolation, so peggysue
	// would populate Position with offsets into that sub-buffer rather than
	// the source file. We zero them to avoid shipping misleading values
	// until a later branch wires command parsing into the main grammar.
	input := "task greet {\n    echo \"hi $NAME\"\n}\n"

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)
	require.Len(t, result.Tasks[0].Commands, 1)

	for _, elem := range result.Tasks[0].Commands[0].Elements {
		switch v := elem.(type) {
		case StringElement:
			require.Equal(t, Position{}, v.Position)
		case VariableElement:
			require.Equal(t, Position{}, v.Position)
		case BacktickElement:
			require.Equal(t, Position{}, v.Position)
		case ExpressionElement:
			require.Equal(t, Position{}, v.Position)
		}
	}
}

func TestPosition_ExpressionInVariable(t *testing.T) {
	input := "GREETING = {{name || \"hello\"}}\n"

	result, ok, err := ParseQuakefile(input)
	require.True(t, ok)
	require.NoError(t, err)
	require.Len(t, result.Variables, 1)

	v := result.Variables[0]
	require.Equal(t, 1, v.Position.Line)

	expr, ok := v.Value.(Or)
	require.True(t, ok, "expected Or expression, got %T", v.Value)

	require.Equal(t, 1, expr.Position.Line)

	left, ok := expr.Left.(Identifier)
	require.True(t, ok, "expected Identifier on left, got %T", expr.Left)
	require.Equal(t, "name", left.Name)
	require.Equal(t, 1, left.Position.Line)
	require.Greater(t, left.Position.End, left.Position.Start)
}
