package parser

import (
	"testing"
)

func TestParseBareEmptyBraces(t *testing.T) {
	input := `task brace {
  {}
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "brace" {
		t.Errorf("Expected task name 'brace', got '%s'", task.Name)
	}

	if len(task.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(task.Commands))
	}

	// The command should be just "{}"
	if len(task.Commands) > 0 {
		cmd := task.Commands[0]
		if len(cmd.Elements) != 1 {
			t.Errorf("Expected 1 element in command, got %d", len(cmd.Elements))
		}
	}
}

func TestParseBareEmptyBracesWithSpaces(t *testing.T) {
	input := `task spaces {
  { }
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "spaces" {
		t.Errorf("Expected task name 'spaces', got '%s'", task.Name)
	}

	if len(task.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(task.Commands))
	}
}

func TestParseBracesInPipeline(t *testing.T) {
	input := `task pipeline {
  echo "test" | {}
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "pipeline" {
		t.Errorf("Expected task name 'pipeline', got '%s'", task.Name)
	}

	if len(task.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(task.Commands))
	}
}

func TestParseBracesInShellGrouping(t *testing.T) {
	input := `task grouping {
  { echo "first"; echo "second"; }
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "grouping" {
		t.Errorf("Expected task name 'grouping', got '%s'", task.Name)
	}

	if len(task.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(task.Commands))
	}
}

func TestParseNestedBracesInConditional(t *testing.T) {
	input := `task conditional {
  if [ -f test ]; then { echo "found"; } fi
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "conditional" {
		t.Errorf("Expected task name 'conditional', got '%s'", task.Name)
	}

	if len(task.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(task.Commands))
	}
}

func TestParseAwkWithBraces(t *testing.T) {
	input := `task awk {
  awk '{ print $1 }' file.txt
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "awk" {
		t.Errorf("Expected task name 'awk', got '%s'", task.Name)
	}

	if len(task.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(task.Commands))
	}
}

func TestParseMultipleBracePatterns(t *testing.T) {
	input := `task complex {
  {}
  { echo "grouped" }
  echo "normal"
  awk '{ print }' 
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "complex" {
		t.Errorf("Expected task name 'complex', got '%s'", task.Name)
	}

	if len(task.Commands) != 4 {
		t.Errorf("Expected 4 commands, got %d", len(task.Commands))
	}
}

func TestParseBracesInQuotes(t *testing.T) {
	// This should already work since quotes are handled separately
	input := `task quoted {
  echo "{}"
  echo '{ test }'
}`

	result, ok, err := parseNoPos(input)
	if !ok || err != nil {
		t.Fatalf("Failed to parse: ok=%v, err=%v", ok, err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(result.Tasks))
	}

	task := result.Tasks[0]
	if task.Name != "quoted" {
		t.Errorf("Expected task name 'quoted', got '%s'", task.Name)
	}

	if len(task.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(task.Commands))
	}
}
