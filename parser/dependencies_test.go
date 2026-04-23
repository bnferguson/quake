package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSingleDependency(t *testing.T) {
	input := `task build => clean {
    echo "Building..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:         "build",
				Dependencies: []string{"clean"},
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

func TestParseMultipleDependencies(t *testing.T) {
	input := `task test => compile, test:prepare {
    echo "Running tests..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:         "test",
				Dependencies: []string{"compile", "test:prepare"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Running tests..."`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseTaskWithArgumentsAndDependencies(t *testing.T) {
	input := `task deploy_env(env) => build, test {
    echo "Deploying to environment: $env"
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:         "deploy_env",
				Arguments:    []string{"env"},
				Dependencies: []string{"build", "test"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Deploying to environment: `},
						VariableElement{Name: "env"},
						StringElement{Value: `"`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseNamespacedTaskNames(t *testing.T) {
	input := `task docs:generate {
    echo "Generating documentation..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name: "docs:generate",
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Generating documentation..."`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseFileDependencies(t *testing.T) {
	input := `task output.txt => input.txt {
    echo "Processing input.txt to create output.txt"
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:         "output.txt",
				Dependencies: []string{"input.txt"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Processing input.txt to create output.txt"`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseComplexDependencyChain(t *testing.T) {
	input := `task clean {
    echo "Cleaning..."
}

task compile => clean {
    echo "Compiling..."
}

task deploy => compile, assets:upload, db:migrate {
    echo "Deploying..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name: "clean",
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Cleaning..."`},
					}},
				},
			},
			{
				Name:         "compile",
				Dependencies: []string{"clean"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Compiling..."`},
					}},
				},
			},
			{
				Name:         "deploy",
				Dependencies: []string{"compile", "assets:upload", "db:migrate"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Deploying..."`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseDependenciesWithSpacing(t *testing.T) {
	input := `task deploy =>  build,   test  ,  assets:upload   {
    echo "Deploying with varied spacing..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:         "deploy",
				Dependencies: []string{"build", "test", "assets:upload"},
				Commands: []Command{
					{Elements: []CommandElement{
						StringElement{Value: `echo "Deploying with varied spacing..."`},
					}},
				},
			},
		},
		Namespaces: []Namespace{},
		Variables:  []Variable{},
	}

	require.Equal(t, expected, result)
}

func TestParseBodylessTasks(t *testing.T) {
	input := `task default => build

task build {
    echo "Building..."
}`

	result, ok, err := parseNoPos(input)
	require.True(t, ok, "parsing should succeed")
	require.NoError(t, err, "should not return error")

	expected := QuakeFile{
		Tasks: []Task{
			{
				Name:         "default",
				Dependencies: []string{"build"},
				Commands:     []Command{}, // Body-less task has empty commands
			},
			{
				Name: "build",
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
