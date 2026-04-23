package parser

import (
	"fmt"
	"strings"

	p "github.com/lab47/peggysue"
)

// Grammar holds all the parsing rules
type Grammar struct {
	quakeFile              p.Rule
	topLevelElement        p.Rule
	comment                p.Rule
	fileNamespaceDirective p.Rule
	variable               p.Rule
	multilineStringVar     p.Rule
	simpleVariable         p.Rule
	variableValue          p.Rule
	commandSubstitution    p.Rule
	expressionValue        p.Rule
	quotedString           p.Rule
	task                   p.Rule
	taskSimple             p.Rule
	taskWithArgs           p.Rule
	taskWithDeps           p.Rule
	taskWithArgsAndDeps    p.Rule
	taskDepsOnly           p.Rule
	taskWithDoc            p.Rule
	namespace              p.Rule
	namespaceRef           p.Rule
	argList                p.Rule
	dependencies           p.Rule
	word                   p.Rule
	ws                     p.Rule
	requiredSpace          p.Rule
	content                p.Rule
	balancedBraceContent   p.Rule
	// Command parsing rules
	commandLine       p.Rule
	commandElement    p.Rule
	commandElements   p.Rule
	plainText         p.Rule
	backtickCmd       p.Rule
	variableRef       p.Rule
	expressionElement p.Rule
	// Expression parsing rules
	expr          p.Rule
	orExpr        p.Rule
	primaryExpr   p.Rule
	accessExpr    p.Rule
	identifier    p.Rule
	stringLiteral p.Rule
}

// NewGrammar creates and initializes a new grammar
func NewGrammar() *Grammar {
	g := &Grammar{}
	g.init()
	return g
}

// init initializes all the grammar rules
func (g *Grammar) init() {
	// Create references first
	namespaceRef := p.R("namespace")
	g.namespaceRef = namespaceRef
	balancedRef := p.R("balancedContent")

	// Define basic rules
	g.ws = p.Star(p.Or(
		p.S(" "),
		p.S("\t"),
		p.S("\n"),
		p.S("\r"),
	))

	g.requiredSpace = p.Plus(p.Or(
		p.S(" "),
		p.S("\t"),
	))

	g.word = p.Transform(
		p.Plus(p.Or(
			p.Range('a', 'z'),
			p.Range('A', 'Z'),
			p.Range('0', '9'),
			p.S("_"),
			p.S(":"),
			p.S("."),
			p.S("-"),
			p.S("/"),
		)),
		func(s string) any {
			return s
		},
	)

	// Define content parsing with balanced braces
	balancedRule := p.Star(p.Or(
		// Double quoted string
		p.Seq(
			p.S("\""),
			p.Star(p.Or(
				p.S("\\\""),
				p.S("\\\\"),
				p.Seq(p.Not(p.S("\"")), p.Any()),
			)),
			p.S("\""),
		),
		// Single quoted string
		p.Seq(
			p.S("'"),
			p.Star(p.Or(
				p.S("\\'"),
				p.S("\\\\"),
				p.Seq(p.Not(p.S("'")), p.Any()),
			)),
			p.S("'"),
		),
		p.S("{}"), // Empty braces
		// Nested braces
		p.Seq(
			p.S("{"),
			balancedRef,
			p.S("}"),
		),
		// Regular character that's not a closing brace
		p.Seq(p.Not(p.S("}")), p.Any()),
	))
	balancedRef.Set(balancedRule)
	g.balancedBraceContent = balancedRule

	g.content = p.Transform(
		g.balancedBraceContent,
		func(s string) any {
			return s
		},
	)

	// Define comment
	// Parse comment and capture its text
	g.comment = p.Transform(
		p.Seq(
			p.S("#"),
			p.Star(p.Or(p.S(" "), p.S("\t"))), // Optional whitespace after #
			p.Star(p.Seq(p.Not(p.S("\n")), p.Any())),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(s string) any {
			// Extract the comment text after the # and optional whitespace
			if len(s) > 0 {
				s = strings.TrimPrefix(s, "#")
				s = strings.TrimSpace(s)
				return s
			}
			return ""
		},
	)

	// Define file namespace directive (unquoted, no opening brace)
	g.fileNamespaceDirective = p.Action(
		p.Seq(
			p.S("namespace"),
			g.requiredSpace,
			p.Named("name", g.word),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			return &FileNamespaceDirective{Name: v.Get("name").(string)}
		},
	)

	// Define expression parsing rules first (needed for variable parsing)
	// Identifier: valid name like "env", "target"
	g.identifier = p.Transform(
		p.Seq(
			p.Or(p.Range('a', 'z'), p.Range('A', 'Z'), p.S("_")),
			p.Star(p.Or(
				p.Range('a', 'z'),
				p.Range('A', 'Z'),
				p.Range('0', '9'),
				p.S("_"),
			)),
		),
		func(s string) any {
			return &Identifier{Name: s}
		},
	)

	// String literal: "text" or 'text'
	g.stringLiteral = p.Or(
		// Double quoted string
		p.Action(
			p.Seq(
				p.S("\""),
				p.Named("content", p.Transform(
					p.Star(p.Or(
						p.S("\\\""),
						p.S("\\\\"),
						p.Seq(p.Not(p.S("\"")), p.Any()),
					)),
					func(s string) any { return s },
				)),
				p.S("\""),
			),
			func(v p.Values) any {
				content := v.Get("content").(string)
				// Unescape the content
				content = strings.ReplaceAll(content, "\\\"", "\"")
				content = strings.ReplaceAll(content, "\\\\", "\\")
				return &StringLiteral{Value: content}
			},
		),
		// Single quoted string
		p.Action(
			p.Seq(
				p.S("'"),
				p.Named("content", p.Transform(
					p.Star(p.Or(
						p.S("\\'"),
						p.S("\\\\"),
						p.Seq(p.Not(p.S("'")), p.Any()),
					)),
					func(s string) any { return s },
				)),
				p.S("'"),
			),
			func(v p.Values) any {
				content := v.Get("content").(string)
				// Unescape the content
				content = strings.ReplaceAll(content, "\\'", "'")
				content = strings.ReplaceAll(content, "\\\\", "\\")
				return &StringLiteral{Value: content}
			},
		),
	)

	// Primary expression: identifier or string literal
	g.primaryExpr = p.Or(g.identifier, g.stringLiteral)

	// Access expression: obj.prop (left-associative)
	g.accessExpr = p.Action(
		p.Seq(
			p.Named("base", g.primaryExpr),
			p.Named("accesses", p.Many(p.Action(
				p.Seq(
					p.S("."),
					p.Named("prop", g.identifier),
				),
				func(v p.Values) any {
					return v.Get("prop").(*Identifier).Name
				},
			), 0, -1, func(values []any) any {
				return values
			})),
		),
		func(v p.Values) any {
			base := exprValue(v.Get("base").(Expression))
			accesses := v.Get("accesses")

			var result Expression = base
			if accesses != nil {
				if accessList, ok := accesses.([]any); ok {
					for _, access := range accessList {
						if prop, ok := access.(string); ok {
							result = &AccessId{Object: result, Property: prop}
						}
					}
				}
			}
			return result
		},
	)

	// Or expression: expr || expr (left-associative)
	g.orExpr = p.Action(
		p.Seq(
			p.Named("left", g.accessExpr),
			p.Named("rights", p.Many(p.Action(
				p.Seq(
					p.Star(p.Or(p.S(" "), p.S("\t"))),
					p.S("||"),
					p.Star(p.Or(p.S(" "), p.S("\t"))),
					p.Named("right", g.accessExpr),
				),
				func(v p.Values) any {
					return exprValue(v.Get("right").(Expression))
				},
			), 0, -1, func(values []any) any {
				return values
			})),
		),
		func(v p.Values) any {
			left := exprValue(v.Get("left").(Expression))
			rights := v.Get("rights")

			var result Expression = left
			if rights != nil {
				if rightList, ok := rights.([]any); ok {
					for _, right := range rightList {
						if rightExpr, ok := right.(Expression); ok {
							result = &Or{Left: result, Right: rightExpr}
						}
					}
				}
			}
			return result
		},
	)

	// Top-level expression
	g.expr = g.orExpr

	// Define variable parsing rules
	g.quotedString = p.Transform(
		p.Seq(
			p.S("\""),
			p.Star(p.Or(
				p.S("\\\""),
				p.S("\\\\"),
				p.Seq(p.Not(p.S("\"")), p.Any()),
			)),
			p.S("\""),
		),
		func(s string) any { return s },
	)

	g.commandSubstitution = p.Action(
		p.Seq(
			p.S("`"),
			p.Named("cmd", p.Transform(
				p.Star(p.Seq(p.Not(p.S("`")), p.Any())),
				func(s string) any { return s },
			)),
			p.S("`"),
		),
		func(v p.Values) any {
			return &Variable{
				Value:               "`" + v.Get("cmd").(string) + "`",
				CommandSubstitution: true,
			}
		},
	)

	g.expressionValue = p.Action(
		p.Seq(
			p.S("{{"),
			p.Named("expr", g.expr),
			p.S("}}"),
		),
		func(v p.Values) any {
			return &Variable{
				Value:        exprValue(v.Get("expr").(Expression)),
				IsExpression: true,
			}
		},
	)

	g.variableValue = p.Or(
		g.commandSubstitution,
		g.expressionValue,
		g.quotedString,
	)

	g.multilineStringVar = p.Action(
		p.Seq(
			p.Named("name", g.word),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("="),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("\"\"\""),
			p.Or(p.S("\n"), p.EOS()),
			p.Named("content", p.Transform(
				p.Star(p.Seq(
					p.Not(p.S("\"\"\"")),
					p.Any(),
				)),
				func(s string) any { return s },
			)),
			p.S("\"\"\""),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			return &Variable{
				Name:        v.Get("name").(string),
				Value:       v.Get("content").(string),
				IsMultiline: true,
			}
		},
	)

	g.simpleVariable = p.Action(
		p.Seq(
			p.Named("name", g.word),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("="),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Named("value", g.variableValue),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			value := v.Get("value")
			switch val := value.(type) {
			case *Variable:
				val.Name = v.Get("name").(string)
				return val
			default:
				return &Variable{
					Name:  v.Get("name").(string),
					Value: val.(string),
				}
			}
		},
	)

	g.variable = p.Or(
		g.multilineStringVar,
		g.simpleVariable,
	)

	// Define argument and dependency parsing
	g.argList = p.Transform(
		p.Star(p.Seq(
			p.Not(p.S(")")),
			p.Any(),
		)),
		func(s string) any {
			return parseArgumentsFromString(s)
		},
	)

	g.dependencies = p.Transform(
		p.Star(p.Seq(
			p.Not(p.Or(p.S("{"), p.S("\n"))),
			p.Any(),
		)),
		func(s string) any {
			return parseDependenciesFromString(s)
		},
	)

	// Define task parsing rules
	g.taskSimple = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return &Task{
				Name:     name,
				Commands: commands,
			}
		},
	)

	g.taskWithArgs = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			p.S("("),
			p.Named("args", g.argList),
			p.S(")"),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			args := v.Get("args").([]string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return &Task{
				Name:      name,
				Arguments: args,
				Commands:  commands,
			}
		},
	)

	g.taskWithDeps = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("=>"),
			g.ws,
			p.Named("deps", g.dependencies),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			deps := v.Get("deps").([]string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return &Task{
				Name:         name,
				Dependencies: deps,
				Commands:     commands,
			}
		},
	)

	g.taskWithArgsAndDeps = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			p.S("("),
			p.Named("args", g.argList),
			p.S(")"),
			g.ws,
			p.S("=>"),
			g.ws,
			p.Named("deps", g.dependencies),
			g.ws,
			p.S("{"),
			p.Named("content", g.content),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			args := v.Get("args").([]string)
			deps := v.Get("deps").([]string)
			content := v.Get("content").(string)
			commands := parseCommands(content)

			return &Task{
				Name:         name,
				Arguments:    args,
				Dependencies: deps,
				Commands:     commands,
			}
		},
	)

	// Task with dependencies only (no body)
	g.taskDepsOnly = p.Action(
		p.Seq(
			p.S("task"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("=>"),
			g.ws,
			p.Named("deps", g.dependencies),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			deps := v.Get("deps").([]string)

			return &Task{
				Name:         name,
				Dependencies: deps,
				Commands:     []Command{}, // Empty commands for deps-only tasks
			}
		},
	)

	g.task = p.Or(
		g.taskWithArgsAndDeps,
		g.taskWithDeps,
		g.taskDepsOnly, // Add this before taskWithArgs to prioritize deps-only parsing
		g.taskWithArgs,
		g.taskSimple,
	)

	// Define namespace rule. Each element is wrapped in positionGuard so the
	// outer Action here (and the Seq(ws, element) action below) doesn't
	// re-stamp the child's Position with a span that includes leading
	// whitespace.
	namespaceRule := p.Action(
		p.Seq(
			p.S("namespace"),
			g.requiredSpace,
			p.Named("name", g.word),
			g.ws,
			p.S("{"),
			p.Named("elements", p.Many(p.Action(
				p.Seq(
					g.ws,
					p.Named("element", p.Or(
						g.comment,
						g.variable,
						g.task,
						g.namespaceRef,
					)),
				),
				func(v p.Values) any {
					return positionGuard{inner: v.Get("element")}
				},
			), 0, -1, func(values []any) any {
				return values
			})),
			p.S("}"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			name := v.Get("name").(string)
			ns := &Namespace{
				Name:       name,
				Tasks:      []Task{},
				Variables:  []Variable{},
				Namespaces: []Namespace{},
			}

			elements := v.Get("elements")
			if elements != nil {
				for _, elem := range elements.([]any) {
					elem = unwrapGuard(elem)
					if elem == nil {
						continue
					}
					switch e := elem.(type) {
					case *Task:
						ns.Tasks = append(ns.Tasks, *e)
					case *Variable:
						ns.Variables = append(ns.Variables, *e)
					case *Namespace:
						ns.Namespaces = append(ns.Namespaces, *e)
					}
				}
			}

			return ns
		},
	)
	// Set the reference using type assertion
	if ref, ok := namespaceRef.(interface{ Set(p.Rule) }); ok {
		ref.Set(namespaceRule)
	}
	g.namespace = namespaceRule

	// Task with optional documentation comment. Wraps the *Task in positionGuard
	// so the outer Action's SetPositioner pass doesn't overwrite the task's
	// Position with a span that includes the preceding doc comment.
	g.taskWithDoc = p.Or(
		// Task with preceding comment
		p.Action(
			p.Seq(
				g.ws,
				p.Named("doc", g.comment),
				g.ws,
				p.Named("task", g.task),
			),
			func(v p.Values) any {
				task := v.Get("task").(*Task)
				if doc, ok := v.Get("doc").(string); ok && doc != "" {
					task.Description = doc
				}
				return positionGuard{inner: task}
			},
		),
		// Task without comment
		g.task,
	)

	// Define top-level element. Wraps the inner value in positionGuard so the
	// surrounding Action's SetPositioner dispatch doesn't overwrite the
	// child's Position with a span that includes leading whitespace.
	g.topLevelElement = p.Action(
		p.Seq(
			g.ws,
			p.Named("element", p.Or(
				g.taskWithDoc, // Try task with doc first
				g.fileNamespaceDirective,
				g.variable,
				g.namespace,
				g.comment, // Standalone comments last
			)),
		),
		func(v p.Values) any {
			return positionGuard{inner: v.Get("element")}
		},
	)

	// Define the main Quakefile rule
	g.quakeFile = p.Action(
		p.Seq(
			p.Named("elements", p.Many(g.topLevelElement, 0, -1, func(values []any) any {
				return values
			})),
			g.ws,
			p.EOS(),
		),
		func(v p.Values) any {
			qf := &QuakeFile{
				Tasks:      []Task{},
				Namespaces: []Namespace{},
				Variables:  []Variable{},
			}

			elements := v.Get("elements")
			if elements != nil {
				if elems, ok := elements.([]any); ok {
					for _, elem := range elems {
						elem = unwrapGuard(elem)
						if elem == nil {
							continue
						}
						switch e := elem.(type) {
						case *Task:
							qf.Tasks = append(qf.Tasks, *e)
						case *Namespace:
							qf.Namespaces = append(qf.Namespaces, *e)
						case *Variable:
							qf.Variables = append(qf.Variables, *e)
						case *FileNamespaceDirective:
							qf.FileNamespace = e.Name
						}
					}
				}
			}

			return qf
		},
	)

	// Define command parsing rules
	// Variable reference: $NAME
	g.variableRef = p.Action(
		p.Seq(
			p.S("$"),
			p.Named("name", p.Transform(
				p.Plus(p.Or(
					p.Range('a', 'z'),
					p.Range('A', 'Z'),
					p.Range('0', '9'),
					p.S("_"),
				)),
				func(s string) any { return s },
			)),
		),
		func(v p.Values) any {
			return &VariableElement{Name: v.Get("name").(string)}
		},
	)

	// Expression: {{expr}}
	g.expressionElement = p.Action(
		p.Seq(
			p.S("{{"),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.Named("expr", g.expr),
			p.Star(p.Or(p.S(" "), p.S("\t"))),
			p.S("}}"),
		),
		func(v p.Values) any {
			return &ExpressionElement{Expression: exprValue(v.Get("expr").(Expression))}
		},
	)

	// Backtick command: `cmd`
	g.backtickCmd = p.Action(
		p.Seq(
			p.S("`"),
			p.Named("cmd", p.Transform(
				p.Star(p.Seq(
					p.Not(p.S("`")),
					p.Any(),
				)),
				func(s string) any { return s },
			)),
			p.S("`"),
		),
		func(v p.Values) any {
			return &BacktickElement{Command: v.Get("cmd").(string)}
		},
	)

	// Plain text that's not a special element
	g.plainText = p.Transform(
		p.Plus(p.Seq(
			p.Not(p.Or(
				p.S("$"),
				p.S("{{"),
				p.S("`"),
				p.S("\n"),
				p.EOS(),
			)),
			p.Any(),
		)),
		func(s string) any {
			return &StringElement{Value: s}
		},
	)

	// A single command element
	g.commandElement = p.Or(
		g.expressionElement,
		g.backtickCmd,
		g.variableRef,
		g.plainText,
	)

	// Command elements (multiple elements). Dereference pointer command
	// elements so the resulting []CommandElement holds value types — keeping
	// the public AST shape stable while letting peggysue mutate positions on
	// the pointer during parsing.
	g.commandElements = p.Many(g.commandElement, 0, -1, func(values []any) any {
		elements := make([]CommandElement, 0, len(values))
		for _, v := range values {
			if elem := cmdElemValue(v); elem != nil {
				elements = append(elements, elem)
			}
		}
		return elements
	})

	// A complete command line
	g.commandLine = p.Action(
		p.Seq(
			p.Named("elements", g.commandElements),
			p.Or(p.S("\n"), p.EOS()),
		),
		func(v p.Values) any {
			elements := v.Get("elements").([]CommandElement)
			return &Command{Elements: elements}
		},
	)

}

// exprValue converts a pointer-typed Expression into its value equivalent.
// Parser actions return pointers so peggysue's SetPositioner can mutate the
// Position field; the public AST exposes value-typed Expressions in struct
// fields, so we dereference at the boundary.
//
// Every Expression implementation with a pointer receiver must be listed
// here. Adding a new Expression type without updating this switch would
// silently leave a pointer inside an interface field, breaking
// reflect.DeepEqual in tests.
func exprValue(e Expression) Expression {
	switch v := e.(type) {
	case *Identifier:
		return *v
	case *StringLiteral:
		return *v
	case *AccessId:
		return *v
	case *Or:
		return *v
	case Identifier, StringLiteral, AccessId, Or:
		return e
	}
	panic(fmt.Sprintf("parser: unhandled Expression type %T in exprValue", e))
}

// zeroCmdElemPosition returns e with its Position (and the Position of any
// nested Expression) cleared. Used by parseCommands, whose element positions
// are offsets into a re-parsed command buffer rather than the source file.
//
// This, exprValue, cmdElemValue, and zeroExprPosition are all exhaustive
// switches over the closed set of CommandElement / Expression types. Adding
// a new type requires a case in each.
func zeroCmdElemPosition(e CommandElement) CommandElement {
	switch v := e.(type) {
	case StringElement:
		v.Position = Position{}
		return v
	case BacktickElement:
		v.Position = Position{}
		return v
	case VariableElement:
		v.Position = Position{}
		return v
	case ExpressionElement:
		v.Position = Position{}
		v.Expression = zeroExprPosition(v.Expression)
		return v
	}
	panic(fmt.Sprintf("parser: unhandled CommandElement type %T in zeroCmdElemPosition", e))
}

// zeroExprPosition returns e with its Position (and any nested Expression
// positions) cleared. See zeroCmdElemPosition.
func zeroExprPosition(e Expression) Expression {
	switch v := e.(type) {
	case Identifier:
		v.Position = Position{}
		return v
	case StringLiteral:
		v.Position = Position{}
		return v
	case AccessId:
		v.Position = Position{}
		v.Object = zeroExprPosition(v.Object)
		return v
	case Or:
		v.Position = Position{}
		v.Left = zeroExprPosition(v.Left)
		v.Right = zeroExprPosition(v.Right)
		return v
	}
	panic(fmt.Sprintf("parser: unhandled Expression type %T in zeroExprPosition", e))
}

// cmdElemValue converts a pointer-typed CommandElement to its value form.
// Returns nil if v is nil. Panics if v is non-nil but not a known
// CommandElement — every implementation must be listed here.
func cmdElemValue(v any) CommandElement {
	if v == nil {
		return nil
	}
	switch e := v.(type) {
	case *StringElement:
		return *e
	case *VariableElement:
		return *e
	case *BacktickElement:
		return *e
	case *ExpressionElement:
		return *e
	case StringElement, VariableElement, BacktickElement, ExpressionElement:
		return v.(CommandElement)
	}
	panic(fmt.Sprintf("parser: unhandled CommandElement type %T in cmdElemValue", v))
}

// ParseQuakefile parses a Quakefile string and returns the AST
func ParseQuakefile(input string) (QuakeFile, bool, error) {
	return ParseQuakefileWithSource(input, "")
}

// ParseQuakefileWithSource parses a Quakefile and tracks the source file.
// The sourceFile is threaded through peggysue so every AST node's Position
// records it, and the legacy Task.SourceFile field is populated for
// backward-compatible consumers.
func ParseQuakefileWithSource(input string, sourceFile string) (QuakeFile, bool, error) {
	parser := p.New()
	grammar := NewGrammar()

	opts := []p.ParseOption{p.WithErrors()}
	if sourceFile != "" {
		opts = append(opts, p.WithFilename(sourceFile))
	}

	result, ok, err := parser.Parse(grammar.quakeFile, input, opts...)

	if !ok || err != nil {
		return QuakeFile{}, ok, err
	}

	if result == nil {
		return QuakeFile{Tasks: []Task{}}, true, nil
	}

	qfPtr := result.(*QuakeFile)
	quakeFile := *qfPtr

	// Set source file for all tasks if provided
	if sourceFile != "" {
		for i := range quakeFile.Tasks {
			quakeFile.Tasks[i].SourceFile = sourceFile
		}
		// Also set for tasks in namespaces
		setNamespaceTaskSourceFile(quakeFile.Namespaces, sourceFile)
	}

	return quakeFile, true, nil
}

// Helper to recursively set source file for namespace tasks
func setNamespaceTaskSourceFile(namespaces []Namespace, sourceFile string) {
	for i := range namespaces {
		for j := range namespaces[i].Tasks {
			namespaces[i].Tasks[j].SourceFile = sourceFile
		}
		setNamespaceTaskSourceFile(namespaces[i].Namespaces, sourceFile)
	}
}

// positionGuard wraps an already-positioned AST pointer so the containing
// Action's SetPositioner dispatch can't overwrite it. Peggysue only calls
// SetPosition on values that implement it, so returning a non-SetPositioner
// wrapper preserves the child's Position even when the outer rule's span
// includes leading whitespace or a preceding doc comment.
type positionGuard struct {
	// inner is always an AST node pointer (*Task, *Namespace, *Variable,
	// *FileNamespaceDirective) or another positionGuard — never a value
	// type and never nil.
	inner any
}

// unwrapGuard peels nested positionGuard wrappers off v. Nesting happens
// for documented tasks: the taskWithDoc Action wraps *Task in a guard, then
// topLevelElement wraps that again — so two layers deep before the QuakeFile
// aggregator sees it.
func unwrapGuard(v any) any {
	for {
		w, ok := v.(positionGuard)
		if !ok {
			return v
		}
		v = w.inner
	}
}

// Helper function to parse commands from content string
func parseCommands(content string) []Command {
	// Create a parser with the command line grammar
	parser := p.New()
	grammar := NewGrammar()

	commands := []Command{}
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], " \t\r")
		if line == "" {
			continue
		}

		// Check for special prefixes
		trimmedLine := strings.TrimSpace(line)
		silent := false
		continueOnError := false

		// Handle special prefixes
		if strings.HasPrefix(trimmedLine, "@") {
			silent = true
			trimmedLine = strings.TrimSpace(trimmedLine[1:])
		} else if strings.HasPrefix(trimmedLine, "-") {
			continueOnError = true
			trimmedLine = strings.TrimSpace(trimmedLine[1:])
		}

		// Check for continuation lines starting with |
		// Accumulate all continuation lines into a single command
		fullCommand := trimmedLine
		for i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(nextLine, "|") {
				// Remove the | prefix and trim leading whitespace
				continuation := strings.TrimSpace(nextLine[1:])
				if continuation != "" {
					fullCommand += " " + continuation
				}
				i++ // Skip this line in the outer loop
			} else {
				break
			}
		}

		// Parse the command line using PEG grammar
		result, ok, _ := parser.Parse(grammar.commandElements, fullCommand, p.WithErrors())

		var elements []CommandElement
		if ok && result != nil {
			if elems, ok := result.([]CommandElement); ok {
				elements = elems
			} else {
				// Fallback to simple string if parsing fails
				elements = []CommandElement{StringElement{Value: fullCommand}}
			}
		} else {
			// If parsing fails, treat the whole line as a string
			elements = []CommandElement{StringElement{Value: fullCommand}}
		}

		// Peggysue populated each element's Position with offsets into
		// fullCommand, not into the source file. Rather than ship misleading
		// values, zero them here. A later branch that wires command parsing
		// into the main grammar can populate real absolute positions.
		for i := range elements {
			elements[i] = zeroCmdElemPosition(elements[i])
		}

		cmd := Command{
			Elements:        elements,
			Silent:          silent,
			ContinueOnError: continueOnError,
		}
		commands = append(commands, cmd)
	}
	return commands
}

// parseArgumentsFromString parses argument string into array
func parseArgumentsFromString(argString string) []string {
	if strings.TrimSpace(argString) == "" {
		return []string{}
	}

	args := []string{}
	parts := strings.Split(argString, ",")
	for _, part := range parts {
		arg := strings.TrimSpace(part)
		if arg != "" {
			args = append(args, arg)
		}
	}
	return args
}

// parseDependenciesFromString parses dependency string into array
func parseDependenciesFromString(depString string) []string {
	depString = strings.TrimSpace(depString)
	if depString == "" {
		return []string{}
	}

	deps := []string{}
	parts := strings.FieldsFunc(depString, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})

	for _, part := range parts {
		if part != "" {
			deps = append(deps, part)
		}
	}
	return deps
}

// Legacy functions kept for compatibility
func parseFileWithFileNamespace() p.Rule {
	return p.Action(
		p.Seq(
			p.Star(p.Or(p.S(" "), p.S("\t"), p.S("\n"), p.S("\r"))),
			p.S("namespace"),
			p.Plus(p.Or(p.S(" "), p.S("\t"))),
			p.Named("namespace", p.Transform(
				p.Plus(p.Or(
					p.Range('a', 'z'),
					p.Range('A', 'Z'),
					p.Range('0', '9'),
					p.S("_"),
				)),
				func(s string) any { return s },
			)),
			p.Star(p.Or(p.S(" "), p.S("\t"), p.S("\n"), p.S("\r"))),
			p.Not(p.S("{")),
			p.Named("tasks", parseTasks()),
			p.Star(p.Or(p.S(" "), p.S("\t"), p.S("\n"), p.S("\r"))),
			p.EOS(),
		),
		func(v p.Values) any {
			namespace := v.Get("namespace").(string)
			tasks := v.Get("tasks").([]Task)

			return QuakeFile{
				FileNamespace: namespace,
				Tasks:         tasks,
			}
		},
	)
}

func parseNamespaceFromContent(name, content string) Namespace {
	content = strings.TrimSpace(content)

	// Must start with {
	if !strings.HasPrefix(content, "{") {
		return Namespace{Name: name}
	}

	// Find the matching closing brace
	braceCount := 0
	var innerContent string

	for i, char := range content {
		switch char {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				// Found the matching closing brace
				innerContent = content[1:i] // Skip opening and closing braces
				break
			}
		}
	}

	tasks := parseTasksFromContent(innerContent)

	return Namespace{
		Name:  name,
		Tasks: tasks,
	}
}

func parseTasks() p.Rule {
	return p.Transform(
		p.Star(p.Any()),
		func(s string) any {
			return parseTasksFromContent(s)
		},
	)
}

func parseTasksFromContent(content string) []Task {
	// This is a temporary implementation
	// In a proper implementation, we'd use the PEG parser recursively
	return []Task{}
}

// parseNamespaceFromLines parses a namespace from lines
func parseNamespaceFromLines(lines []string, startIndex int) (*Namespace, int) {
	// This is a temporary implementation
	// In a proper implementation, we'd use the PEG parser recursively
	return nil, startIndex + 1
}
