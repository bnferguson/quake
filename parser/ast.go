package parser

import (
	"encoding/json"
	"fmt"
)

// Position records where an AST node was parsed from, suitable for
// editor tooling (diagnostics, go-to-definition, hover).
//
// Start and End are byte offsets into the source. End is exclusive.
// Line is 1-indexed. Filename is the logical source path passed to
// ParseQuakefileWithSource (empty when parsing a raw string).
//
// Positions are not serialized to JSON, so they do not round-trip.
//
// Top-level nodes (Task, Namespace, Variable, FileNamespaceDirective,
// QuakeFile) carry accurate absolute source offsets. Command-line
// elements (StringElement, VariableElement, ExpressionElement,
// BacktickElement) have their Position zeroed today because
// parseCommands re-parses each command line in isolation; a later
// branch that inlines command parsing into the main grammar can
// populate them properly.
type Position struct {
	Start    int    `json:"-"`
	End      int    `json:"-"`
	Line     int    `json:"-"`
	Filename string `json:"-"`
}

// QuakeFile represents the root of a parsed Quakefile
type QuakeFile struct {
	Tasks         []Task      `json:"tasks"`
	Namespaces    []Namespace `json:"namespaces,omitempty"`
	Variables     []Variable  `json:"variables,omitempty"`
	FileNamespace string      `json:"file_namespace,omitempty"`
	Position      Position    `json:"-"`
}

// SetPosition implements peggysue.SetPositioner.
func (q *QuakeFile) SetPosition(start, end, line int, filename string) {
	q.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// UnmarshalJSON ensures empty slices are initialized correctly
func (q *QuakeFile) UnmarshalJSON(data []byte) error {
	type Alias QuakeFile
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(q),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	// Initialize nil slices to empty slices
	if q.Tasks == nil {
		q.Tasks = []Task{}
	}
	if q.Namespaces == nil {
		q.Namespaces = []Namespace{}
	}
	if q.Variables == nil {
		q.Variables = []Variable{}
	}
	return nil
}

// Task represents a task definition in a Quakefile
type Task struct {
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Arguments    []string  `json:"arguments,omitempty"`
	Dependencies []string  `json:"dependencies,omitempty"`
	Commands     []Command `json:"commands"`
	IsGoTask     bool      `json:"is_go_task,omitempty"`
	GoDispatcher string    `json:"go_dispatcher,omitempty"` // Path to dispatcher main.go
	GoSourceDir  string    `json:"go_source_dir,omitempty"` // Directory containing Go sources
	SourceFile   string    `json:"source_file,omitempty"`   // Source file where task is defined
	Position     Position  `json:"-"`
}

// SetPosition implements peggysue.SetPositioner.
func (t *Task) SetPosition(start, end, line int, filename string) {
	t.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// Variable represents a variable assignment
type Variable struct {
	Name                string   `json:"name"`
	Value               any      `json:"value"` // Can be string, Expression, or BacktickElement
	IsExpression        bool     `json:"is_expression,omitempty"`
	CommandSubstitution bool     `json:"command_substitution,omitempty"`
	IsMultiline         bool     `json:"is_multiline,omitempty"`
	Position            Position `json:"-"`
}

// SetPosition implements peggysue.SetPositioner.
func (v *Variable) SetPosition(start, end, line int, filename string) {
	v.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// Namespace represents a namespace block containing tasks and nested namespaces
type Namespace struct {
	Name       string      `json:"name"`
	Tasks      []Task      `json:"tasks,omitempty"`
	Variables  []Variable  `json:"variables,omitempty"`
	Namespaces []Namespace `json:"namespaces,omitempty"`
	Position   Position    `json:"-"`
}

// SetPosition implements peggysue.SetPositioner.
func (n *Namespace) SetPosition(start, end, line int, filename string) {
	n.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// Command represents a single command line in a task
type Command struct {
	Elements        []CommandElement `json:"elements"`
	Silent          bool             `json:"silent,omitempty"`
	ContinueOnError bool             `json:"continue_on_error,omitempty"`
	Position        Position         `json:"-"`
}

// SetPosition implements peggysue.SetPositioner.
func (c *Command) SetPosition(start, end, line int, filename string) {
	c.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// CommandElement represents a part of a command
type CommandElement interface {
	commandElement()
}

// StringElement represents a literal string in a command
type StringElement struct {
	Value    string   `json:"value"`
	Position Position `json:"-"`
}

func (StringElement) commandElement() {}

// SetPosition implements peggysue.SetPositioner.
func (s *StringElement) SetPosition(start, end, line int, filename string) {
	s.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// BacktickElement represents a command substitution
type BacktickElement struct {
	Command  string   `json:"command"`
	Position Position `json:"-"`
}

func (BacktickElement) commandElement() {}

// SetPosition implements peggysue.SetPositioner.
func (b *BacktickElement) SetPosition(start, end, line int, filename string) {
	b.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// ExpressionElement represents an expression like {{expr}}
type ExpressionElement struct {
	Expression Expression `json:"expression"`
	Position   Position   `json:"-"`
}

func (ExpressionElement) commandElement() {}

// SetPosition implements peggysue.SetPositioner.
func (e *ExpressionElement) SetPosition(start, end, line int, filename string) {
	e.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// VariableElement represents a variable reference like $VAR
type VariableElement struct {
	Name     string   `json:"name"`
	Position Position `json:"-"`
}

func (VariableElement) commandElement() {}

// SetPosition implements peggysue.SetPositioner.
func (v *VariableElement) SetPosition(start, end, line int, filename string) {
	v.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// Expression AST nodes for parsing inside {{}} blocks
type Expression interface {
	expression()
}

// Identifier represents a simple identifier like "env" or "target"
type Identifier struct {
	Name     string   `json:"name"`
	Position Position `json:"-"`
}

func (Identifier) expression() {}

// SetPosition implements peggysue.SetPositioner.
func (i *Identifier) SetPosition(start, end, line int, filename string) {
	i.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// AccessId represents dot notation like "env.API_KEY"
type AccessId struct {
	Object   Expression `json:"object"`
	Property string     `json:"property"`
	Position Position   `json:"-"`
}

func (AccessId) expression() {}

// SetPosition implements peggysue.SetPositioner.
func (a *AccessId) SetPosition(start, end, line int, filename string) {
	a.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// StringLiteral represents a quoted string in expressions
type StringLiteral struct {
	Value    string   `json:"value"`
	Position Position `json:"-"`
}

func (StringLiteral) expression() {}

// SetPosition implements peggysue.SetPositioner.
func (s *StringLiteral) SetPosition(start, end, line int, filename string) {
	s.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// FileNamespaceDirective represents a file-level namespace directive,
// written as `namespace foo` at the top of a file (no opening brace).
type FileNamespaceDirective struct {
	Name     string
	Position Position
}

// SetPosition implements peggysue.SetPositioner.
func (f *FileNamespaceDirective) SetPosition(start, end, line int, filename string) {
	f.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// Or represents the || operator
type Or struct {
	Left     Expression `json:"left"`
	Right    Expression `json:"right"`
	Position Position   `json:"-"`
}

func (Or) expression() {}

// SetPosition implements peggysue.SetPositioner.
func (o *Or) SetPosition(start, end, line int, filename string) {
	o.Position = Position{Start: start, End: end, Line: line, Filename: filename}
}

// MarshalJSON for Expression interface
func marshalExpression(expr Expression) (any, error) {
	switch e := expr.(type) {
	case Identifier:
		return struct {
			Type string `json:"type"`
			Name string `json:"name"`
		}{"identifier", e.Name}, nil
	case AccessId:
		obj, err := marshalExpression(e.Object)
		if err != nil {
			return nil, err
		}
		return struct {
			Type     string `json:"type"`
			Object   any    `json:"object"`
			Property string `json:"property"`
		}{"access", obj, e.Property}, nil
	case StringLiteral:
		return struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		}{"string", e.Value}, nil
	case Or:
		left, err := marshalExpression(e.Left)
		if err != nil {
			return nil, err
		}
		right, err := marshalExpression(e.Right)
		if err != nil {
			return nil, err
		}
		return struct {
			Type  string `json:"type"`
			Left  any    `json:"left"`
			Right any    `json:"right"`
		}{"or", left, right}, nil
	default:
		return nil, fmt.Errorf("unknown expression type: %T", e)
	}
}

// MarshalJSON for Command to handle the interface slice
func (c Command) MarshalJSON() ([]byte, error) {
	// Create concrete types with type tags for marshaling
	elements := make([]any, len(c.Elements))
	for i, elem := range c.Elements {
		switch e := elem.(type) {
		case StringElement:
			elements[i] = struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			}{"string", e.Value}
		case BacktickElement:
			elements[i] = struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			}{"backtick", e.Command}
		case ExpressionElement:
			expr, err := marshalExpression(e.Expression)
			if err != nil {
				return nil, err
			}
			elements[i] = struct {
				Type       string `json:"type"`
				Expression any    `json:"expression"`
			}{"expression", expr}
		case VariableElement:
			elements[i] = struct {
				Type string `json:"type"`
				Name string `json:"name"`
			}{"variable", e.Name}
		default:
			return nil, fmt.Errorf("unknown command element type: %T", e)
		}
	}

	return json.Marshal(struct {
		Elements        []any `json:"elements"`
		Silent          bool  `json:"silent,omitempty"`
		ContinueOnError bool  `json:"continue_on_error,omitempty"`
	}{
		Elements:        elements,
		Silent:          c.Silent,
		ContinueOnError: c.ContinueOnError,
	})
}

// UnmarshalJSON for Command to handle the interface slice
func (c *Command) UnmarshalJSON(data []byte) error {
	var temp struct {
		Elements        []json.RawMessage `json:"elements"`
		Silent          bool              `json:"silent,omitempty"`
		ContinueOnError bool              `json:"continue_on_error,omitempty"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	c.Silent = temp.Silent
	c.ContinueOnError = temp.ContinueOnError
	c.Elements = make([]CommandElement, 0, len(temp.Elements))

	for _, raw := range temp.Elements {
		var typeCheck struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeCheck); err != nil {
			return err
		}

		switch typeCheck.Type {
		case "string":
			var elem StringElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		case "backtick":
			var elem BacktickElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		case "expression":
			var elem ExpressionElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		case "variable":
			var elem VariableElement
			if err := json.Unmarshal(raw, &elem); err != nil {
				return err
			}
			c.Elements = append(c.Elements, elem)
		default:
			return fmt.Errorf("unknown command element type: %s", typeCheck.Type)
		}
	}

	return nil
}

// MarshalJSON for Variable to handle different value types
func (v Variable) MarshalJSON() ([]byte, error) {
	var value any
	var err error

	switch val := v.Value.(type) {
	case string:
		value = val
	case Expression:
		value, err = marshalExpression(val)
		if err != nil {
			return nil, err
		}
	case BacktickElement:
		value = struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		}{"backtick", val.Command}
	default:
		value = val
	}

	return json.Marshal(struct {
		Name                string `json:"name"`
		Value               any    `json:"value"`
		IsExpression        bool   `json:"is_expression,omitempty"`
		CommandSubstitution bool   `json:"command_substitution,omitempty"`
		IsMultiline         bool   `json:"is_multiline,omitempty"`
	}{
		Name:                v.Name,
		Value:               value,
		IsExpression:        v.IsExpression,
		CommandSubstitution: v.CommandSubstitution,
		IsMultiline:         v.IsMultiline,
	})
}
