package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseTaskDocumentation(t *testing.T) {
	input := `# Build the application
task build {
    echo "Building..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:        "build",
				Description: "Build the application",
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Building..."`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseMultipleTasksWithDocumentation(t *testing.T) {
	input := `# Clean build artifacts
task clean {
    rm -rf build/
}

# Compile the source code
task compile {
    go build
}

# Run all tests
task test {
    go test ./...
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	require.Len(t, result.Tasks, 3)

	require.Equal(t, "clean", result.Tasks[0].Name)
	require.Equal(t, "Clean build artifacts", result.Tasks[0].Description)

	require.Equal(t, "compile", result.Tasks[1].Name)
	require.Equal(t, "Compile the source code", result.Tasks[1].Description)

	require.Equal(t, "test", result.Tasks[2].Name)
	require.Equal(t, "Run all tests", result.Tasks[2].Description)
}

func TestParseTaskWithoutDocumentation(t *testing.T) {
	input := `task build {
    echo "Building..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:        "build",
				Description: "", // No description
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Building..."`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseMixedDocumentedAndUndocumented(t *testing.T) {
	input := `# Clean the project
task clean {
    rm -rf build/
}

task helper {
    echo "Helper task"
}

# Main build task
task build {
    make all
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	require.Len(t, result.Tasks, 3)

	require.Equal(t, "Clean the project", result.Tasks[0].Description)
	require.Equal(t, "", result.Tasks[1].Description)
	require.Equal(t, "Main build task", result.Tasks[2].Description)
}

func TestParseTaskDocumentationWithDependencies(t *testing.T) {
	input := `# Build everything
task build => clean, compile {
    echo "Build complete"
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	require.Len(t, result.Tasks, 1)
	task := result.Tasks[0]

	require.Equal(t, "build", task.Name)
	require.Equal(t, "Build everything", task.Description)
	require.Equal(t, []string{"clean", "compile"}, task.Dependencies)
}

func TestParseBodylessTaskWithDocumentation(t *testing.T) {
	input := `# Default task that runs build
task default => build`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	require.Len(t, result.Tasks, 1)
	task := result.Tasks[0]

	require.Equal(t, "default", task.Name)
	require.Equal(t, "Default task that runs build", task.Description)
	require.Equal(t, []string{"build"}, task.Dependencies)
	require.Empty(t, task.Commands)
}

func TestParseStandaloneComments(t *testing.T) {
	input := `# This is a standalone comment
# Another standalone comment

# Task documentation
task build {
    echo "Building..."
}

# More standalone comments
# at the end`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	// Should only have one task with its documentation
	require.Len(t, result.Tasks, 1)
	require.Equal(t, "build", result.Tasks[0].Name)
	require.Equal(t, "Task documentation", result.Tasks[0].Description)
}
