package parser

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBasicQuakefile(t *testing.T) {
	// Read the actual basic Quakefile for future use
	_, err := os.ReadFile("../testdata/basic/Quakefile")
	require.NoError(t, err, "should read basic Quakefile")

	// For now, just test the first task manually since our current parser only handles one task
	// This is the first step - later we'll extend to handle multiple tasks
	firstTaskContent := `task default {
    echo "Running default task"
    echo "This is the default action"
}`

	result, ok, err := parseNoPos(firstTaskContent)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	// Check the structure
	require.Len(t, result.Tasks, 1, "should have one task")
	require.Equal(t, "default", result.Tasks[0].Name, "task name should be default")
	require.Len(t, result.Tasks[0].Commands, 2, "should have two commands")
	// Check command elements
	require.Len(t, result.Tasks[0].Commands[0].Elements, 1)
	require.Len(t, result.Tasks[0].Commands[1].Elements, 1)

	// Test JSON serialization
	jsonData, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err, "should serialize to JSON")

	t.Logf("Generated JSON:\n%s", jsonData)

	// Save to expected output for comparison
	err = os.WriteFile("../testdata/basic/expected_ast.json", jsonData, 0644)
	require.NoError(t, err, "should write expected JSON")
}

func TestParseHelloTask(t *testing.T) {
	// Test the hello task specifically
	helloTaskContent := `task hello {
    echo "Hello, World!"
}`

	result, ok, err := parseNoPos(helloTaskContent)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name: "hello",
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: "echo \"Hello, World!\""},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseGreetPersonTask(t *testing.T) {
	// Test task with arguments
	greetTaskContent := `task greet_person(name) {
    echo "Hello, $name!"
    echo "Nice to meet you"
}`

	result, ok, err := parseNoPos(greetTaskContent)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:      "greet_person",
				Arguments: []string{"name"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: "echo \"Hello, "},
						VariableElement{Name: "name"},
						StringElement{Value: "!\""},
					}},
					{Elements: []CommandElement{
						StringElement{Value: "echo \"Nice to meet you\""},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}
