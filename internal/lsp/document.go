package lsp

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"

	"miren.dev/quake/analysis"
	"miren.dev/quake/parser"
)

// document is the server's view of one open Quakefile. It owns the
// text, its parse result, and every derived index a handler might
// need. Handlers receive a fresh document whenever the client sends
// didOpen or didChange; replacing rather than mutating keeps the
// read path lock-free after the swap.
type document struct {
	uri     string
	version int32
	text    string
	lines   *lineIndex

	// quakeFile is nil when the parser failed. diagnostics still
	// reports a single entry in that case so the client has
	// something to surface.
	quakeFile *parser.QuakeFile

	// symbols is nil when quakeFile is nil. Every handler that needs
	// to resolve a name guards on a nil document first.
	symbols *analysis.SymbolTable

	// parseErr and parseOk capture the peggysue return shape so
	// diagnostics() can distinguish "unknown parse failure" from a
	// wrapped I/O-style error.
	parseErr error
	parseOk  bool
}

// parse reads text and returns a fully-populated document. It never
// errors — parse failures become diagnostics.
func parse(uri, text string, version int32) *document {
	d := &document{
		uri:     uri,
		version: version,
		text:    text,
		lines:   newLineIndex(text),
	}

	qf, ok, err := parser.ParseQuakefileWithSource(text, uriToPath(uri))
	d.parseOk = ok
	d.parseErr = err
	if err == nil && ok {
		d.quakeFile = &qf
		d.symbols = analysis.BuildSymbolTable(&qf)
	}
	return d
}

// diagnosticSource is the value reported on every published
// Diagnostic. Clients show this as a prefix ("quake:") so users can
// tell our diagnostics apart from another server's.
const diagnosticSource = "quake"

// diagnostics returns the full LSP diagnostic set for the document:
// a synthetic entry for parse failure, plus every structural problem
// analysis.Diagnose finds when the parse succeeded.
func (d *document) diagnostics() []protocol.Diagnostic {
	source := diagnosticSource
	if d.parseErr != nil {
		return []protocol.Diagnostic{{
			Range:    protocol.Range{},
			Severity: severityPtr(protocol.DiagnosticSeverityError),
			Source:   &source,
			Message:  fmt.Sprintf("parse error: %v", d.parseErr),
		}}
	}
	if !d.parseOk {
		return []protocol.Diagnostic{{
			Range:    protocol.Range{},
			Severity: severityPtr(protocol.DiagnosticSeverityError),
			Source:   &source,
			Message:  "unknown parse failure",
		}}
	}

	analysisDiags := analysis.DiagnoseWith(d.quakeFile, d.symbols)
	if len(analysisDiags) == 0 {
		return nil
	}

	out := make([]protocol.Diagnostic, 0, len(analysisDiags))
	for _, diag := range analysisDiags {
		out = append(out, protocol.Diagnostic{
			Range:    d.lines.rangeOf(diag.Position),
			Severity: severityPtr(toLSPSeverity(diag.Severity)),
			Source:   &source,
			Message:  diag.Message,
		})
	}
	return out
}

// documentSymbols returns a hierarchical outline for the current
// file: top-level tasks and variables, then a nested entry per
// namespace with its own children.
func (d *document) documentSymbols() []protocol.DocumentSymbol {
	if d.quakeFile == nil {
		return nil
	}

	var out []protocol.DocumentSymbol
	for i := range d.quakeFile.Tasks {
		out = append(out, d.taskSymbol(&d.quakeFile.Tasks[i]))
	}
	for i := range d.quakeFile.Variables {
		out = append(out, d.variableSymbol(&d.quakeFile.Variables[i]))
	}
	for i := range d.quakeFile.Namespaces {
		out = append(out, d.namespaceSymbol(&d.quakeFile.Namespaces[i]))
	}
	return out
}

// definition returns the location of the symbol referenced at pos, or
// nil if no identifier sits under the cursor or the name is
// undefined. The lookup is intentionally forgiving: it resolves a
// qualified identifier (like `db:migrate`) regardless of which side
// of the colon the cursor is on, and tries both task and variable
// tables so the handler does not need to know which context it is in.
func (d *document) definition(pos protocol.Position) *protocol.Location {
	if d.symbols == nil {
		return nil
	}
	offset := int(pos.IndexIn(d.text))
	name := qualifiedNameAt(d.text, offset)
	if name == "" {
		return nil
	}
	if task := d.symbols.Task(name); task != nil {
		return d.locationOf(task.Position)
	}
	if v := d.symbols.Variable(name); v != nil {
		return d.locationOf(v.Position)
	}
	if ns := d.symbols.Namespace(name); ns != nil {
		return d.locationOf(ns.Position)
	}
	return nil
}

func (d *document) taskSymbol(t *parser.Task) protocol.DocumentSymbol {
	r := d.lines.rangeOf(t.Position)
	return protocol.DocumentSymbol{
		Name:           t.Name,
		Detail:         taskDetail(t),
		Kind:           protocol.SymbolKindFunction,
		Range:          r,
		SelectionRange: r,
	}
}

func (d *document) variableSymbol(v *parser.Variable) protocol.DocumentSymbol {
	r := d.lines.rangeOf(v.Position)
	return protocol.DocumentSymbol{
		Name:           v.Name,
		Kind:           protocol.SymbolKindVariable,
		Range:          r,
		SelectionRange: r,
	}
}

func (d *document) namespaceSymbol(n *parser.Namespace) protocol.DocumentSymbol {
	r := d.lines.rangeOf(n.Position)

	var children []protocol.DocumentSymbol
	for i := range n.Tasks {
		children = append(children, d.taskSymbol(&n.Tasks[i]))
	}
	for i := range n.Variables {
		children = append(children, d.variableSymbol(&n.Variables[i]))
	}
	for i := range n.Namespaces {
		children = append(children, d.namespaceSymbol(&n.Namespaces[i]))
	}

	return protocol.DocumentSymbol{
		Name:           n.Name,
		Kind:           protocol.SymbolKindNamespace,
		Range:          r,
		SelectionRange: r,
		Children:       children,
	}
}

// locationOf converts a parser.Position to an LSP Location. A node
// from the same file reuses d.uri unchanged; only a foreign filename
// triggers a pathToURI conversion. Comparison is done in URI space
// so percent-encoded paths (spaces, non-ASCII) match canonically.
func (d *document) locationOf(pos parser.Position) *protocol.Location {
	uri := d.uri
	if pos.Filename != "" {
		if other := pathToURI(pos.Filename); other != d.uri {
			uri = other
		}
	}
	return &protocol.Location{
		URI:   uri,
		Range: d.lines.rangeOf(pos),
	}
}

// taskDetail renders the task's argument list as a short signature
// string, suitable for the Detail field of a DocumentSymbol. Returns
// nil when there are no arguments so clients render a bare name.
func taskDetail(t *parser.Task) *string {
	if len(t.Arguments) == 0 {
		return nil
	}
	s := "("
	for i, a := range t.Arguments {
		if i > 0 {
			s += ", "
		}
		s += a
	}
	s += ")"
	return &s
}

// toLSPSeverity maps an analysis severity into its LSP counterpart.
// The default arm degrades unknown values to Information rather than
// dropping the diagnostic — extend the switch when analysis grows a
// new level.
func toLSPSeverity(s analysis.Severity) protocol.DiagnosticSeverity {
	switch s {
	case analysis.SeverityError:
		return protocol.DiagnosticSeverityError
	case analysis.SeverityWarning:
		return protocol.DiagnosticSeverityWarning
	default:
		return protocol.DiagnosticSeverityInformation
	}
}

func severityPtr(s protocol.DiagnosticSeverity) *protocol.DiagnosticSeverity {
	return &s
}
