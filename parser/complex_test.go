package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseComplexQuakefile(t *testing.T) {
	// Read the complex test data
	quakefilePath := filepath.Join("..", "testdata", "complex", "Quakefile")
	expectedPath := filepath.Join("..", "testdata", "complex", "expected_ast.json")

	inputData, err := os.ReadFile(quakefilePath)
	require.NoError(t, err, "should read complex Quakefile")

	expectedData, err := os.ReadFile(expectedPath)
	require.NoError(t, err, "should read expected AST")

	// Parse the Quakefile
	result, ok, err := parseNoPos(string(inputData))
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	// Compare JSON representations to handle Expression interfaces correctly
	actualJSON, err := json.Marshal(result)
	require.NoError(t, err, "should marshal actual result")

	// Normalize both JSON by unmarshaling and remarshaling
	var expectedNormalized, actualNormalized map[string]interface{}
	err = json.Unmarshal(expectedData, &expectedNormalized)
	require.NoError(t, err, "should unmarshal expected JSON")

	err = json.Unmarshal(actualJSON, &actualNormalized)
	require.NoError(t, err, "should unmarshal actual JSON")

	require.Equal(t, expectedNormalized, actualNormalized, "parsed result should match expected AST")

	// Additional specific assertions for the complex file
	require.Len(t, result.Tasks, 21, "should have 21 top-level tasks")
	require.Len(t, result.Variables, 12, "should have 12 global variables")
	require.Len(t, result.Namespaces, 3, "should have 3 namespaces")

	// Check that we have the expected namespaces
	namespaceNames := make(map[string]bool)
	for _, ns := range result.Namespaces {
		namespaceNames[ns.Name] = true
	}
	require.True(t, namespaceNames["docker"], "should have docker namespace")
	require.True(t, namespaceNames["db"], "should have db namespace")
	require.True(t, namespaceNames["ci"], "should have ci namespace")

	// Check variable types
	variableTypes := make(map[string]string)
	for _, v := range result.Variables {
		if v.IsExpression {
			variableTypes[v.Name] = "expression"
		} else if v.CommandSubstitution {
			variableTypes[v.Name] = "command"
		} else {
			variableTypes[v.Name] = "string"
		}
	}

	// Verify specific variable types
	require.Equal(t, "string", variableTypes["PROJECT"])
	require.Equal(t, "string", variableTypes["BINARY"])
	require.Equal(t, "command", variableTypes["VERSION"])
	require.Equal(t, "command", variableTypes["BUILD_TIME"])
	require.Equal(t, "command", variableTypes["GIT_COMMIT"])
	require.Equal(t, "command", variableTypes["GO_VERSION"])

	// Check for specific tasks with dependencies
	taskDeps := make(map[string][]string)
	for _, task := range result.Tasks {
		if len(task.Dependencies) > 0 {
			taskDeps[task.Name] = task.Dependencies
		}
	}

	// Verify key dependency relationships
	require.Equal(t, []string{"build"}, taskDeps["default"])
	require.Equal(t, []string{"fmt"}, taskDeps["lint"])
	require.Equal(t, []string{"deps"}, taskDeps["test"])
	require.Equal(t, []string{"deps", "generate", "lint"}, taskDeps["build"])
	require.Equal(t, []string{"build:all"}, taskDeps["dist"])
	require.Equal(t, []string{"build"}, taskDeps["run"])

	// Check namespace tasks have proper structure
	dockerNamespace := result.Namespaces[0] // Assuming docker is first
	if dockerNamespace.Name != "docker" {
		// Find docker namespace
		for _, ns := range result.Namespaces {
			if ns.Name == "docker" {
				dockerNamespace = ns
				break
			}
		}
	}
	require.Equal(t, "docker", dockerNamespace.Name)
	require.Len(t, dockerNamespace.Tasks, 3, "docker namespace should have 3 tasks")
	require.Len(t, dockerNamespace.Variables, 3, "docker namespace should have 3 variables")
}
