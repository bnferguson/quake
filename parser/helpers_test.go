package parser

// stripPositions zeros every Position field in a QuakeFile tree so tests can
// compare against expected structs built without position information.
func stripPositions(qf *QuakeFile) {
	qf.Position = Position{}
	for i := range qf.Tasks {
		stripTaskPositions(&qf.Tasks[i])
	}
	for i := range qf.Namespaces {
		stripNamespacePositions(&qf.Namespaces[i])
	}
	for i := range qf.Variables {
		stripVariablePositions(&qf.Variables[i])
	}
}

func stripTaskPositions(t *Task) {
	t.Position = Position{}
	for i := range t.Commands {
		stripCommandPositions(&t.Commands[i])
	}
}

func stripNamespacePositions(n *Namespace) {
	n.Position = Position{}
	for i := range n.Tasks {
		stripTaskPositions(&n.Tasks[i])
	}
	for i := range n.Namespaces {
		stripNamespacePositions(&n.Namespaces[i])
	}
	for i := range n.Variables {
		stripVariablePositions(&n.Variables[i])
	}
}

func stripVariablePositions(v *Variable) {
	v.Position = Position{}
	if expr, ok := v.Value.(Expression); ok {
		v.Value = stripExprPositions(expr)
	}
}

func stripCommandPositions(c *Command) {
	c.Position = Position{}
	for i, elem := range c.Elements {
		c.Elements[i] = stripCommandElementPositions(elem)
	}
}

func stripCommandElementPositions(e CommandElement) CommandElement {
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
		v.Expression = stripExprPositions(v.Expression)
		return v
	}
	return e
}

func stripExprPositions(e Expression) Expression {
	switch v := e.(type) {
	case Identifier:
		v.Position = Position{}
		return v
	case StringLiteral:
		v.Position = Position{}
		return v
	case AccessId:
		v.Position = Position{}
		v.Object = stripExprPositions(v.Object)
		return v
	case Or:
		v.Position = Position{}
		v.Left = stripExprPositions(v.Left)
		v.Right = stripExprPositions(v.Right)
		return v
	}
	return e
}

// parseNoPos parses input and strips all positions — useful in tests that
// compare against expected literals built without Position data.
func parseNoPos(input string) (QuakeFile, bool, error) {
	return parseNoPosWithSource(input, "")
}

func parseNoPosWithSource(input, source string) (QuakeFile, bool, error) {
	result, ok, err := ParseQuakefileWithSource(input, source)
	if ok {
		stripPositions(&result)
	}
	return result, ok, err
}
