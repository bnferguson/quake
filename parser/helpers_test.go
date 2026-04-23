package parser

// stripPositions zeros every Position field in a QuakeFile tree so tests can
// compare against expected structs built without position information. Leaf
// clearing delegates to the production zeroExprPosition and
// zeroCmdElemPosition so a new AST type only needs to be listed in one
// place.
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
		v.Value = zeroExprPosition(expr)
	}
}

func stripCommandPositions(c *Command) {
	c.Position = Position{}
	for i, elem := range c.Elements {
		c.Elements[i] = zeroCmdElemPosition(elem)
	}
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
