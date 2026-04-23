package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBacktickCommands(t *testing.T) {
	input := `task deploy {
    echo "Current commit:"
    ` + "`" + `git rev-parse --short HEAD` + "`" + `
    echo "Current date:"
    ` + "`" + `date +%Y-%m-%d` + "`" + `
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "deploy",
			Commands: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "echo \"Current commit:\""},
				}},
				{Elements: []CommandElement{
					BacktickElement{Command: "git rev-parse --short HEAD"},
				}},
				{Elements: []CommandElement{
					StringElement{Value: "echo \"Current date:\""},
				}},
				{Elements: []CommandElement{
					BacktickElement{Command: "date +%Y-%m-%d"},
				}},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseBacktickWithPrefixes(t *testing.T) {
	input := `task info {
    @` + "`" + `echo "Silent command"` + "`" + `
    -` + "`" + `false || true` + "`" + `
    ` + "`" + `pwd` + "`" + `
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "info",
			Commands: []Command{
				{
					Elements: []CommandElement{
						BacktickElement{Command: "echo \"Silent command\""},
					},
					Silent: true,
				},
				{
					Elements: []CommandElement{
						BacktickElement{Command: "false || true"},
					},
					ContinueOnError: true,
				},
				{
					Elements: []CommandElement{
						BacktickElement{Command: "pwd"},
					},
				},
			},
		},
	}

	require.Equal(t, expected, result)
}

func TestParseMixedCommandsAndBackticks(t *testing.T) {
	input := `task build {
    echo "Building..."
    ` + "`" + `make clean` + "`" + `
    make build
    ` + "`" + `make test` + "`" + `
    echo "Done"
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := makeQuakeFile()
	expected.Tasks = []Task{
		{
			Name: "build",
			Commands: []Command{
				{Elements: []CommandElement{
					StringElement{Value: "echo \"Building...\""},
				}},
				{Elements: []CommandElement{
					BacktickElement{Command: "make clean"},
				}},
				{Elements: []CommandElement{
					StringElement{Value: "make build"},
				}},
				{Elements: []CommandElement{
					BacktickElement{Command: "make test"},
				}},
				{Elements: []CommandElement{
					StringElement{Value: "echo \"Done\""},
				}},
			},
		},
	}

	require.Equal(t, expected, result)
}
