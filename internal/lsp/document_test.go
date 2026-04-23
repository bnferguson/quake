package lsp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const testURI = "file:///tmp/Quakefile"

func TestDocument_DiagnosticsFromParseFailure(t *testing.T) {
	// "task foo" with no body — peggysue reports a parse failure.
	d := parse(testURI, "task foo\n", 1)

	diags := d.diagnostics()
	require.Len(t, diags, 1)
	require.Equal(t, severityPtr(protocol.DiagnosticSeverityError), diags[0].Severity)
	require.NotEmpty(t, diags[0].Message)
}

func TestDocument_DiagnosticsFromAnalysis(t *testing.T) {
	// Dependency on an undefined task — analysis should catch it,
	// the parser will not.
	src := `
task build => missing {
    echo hi
}
`
	d := parse(testURI, src, 1)

	diags := d.diagnostics()
	require.NotEmpty(t, diags, "undefined dependency should produce a diagnostic")

	var found bool
	for _, diag := range diags {
		if diag.Message == `task "build" depends on undefined task "missing"` {
			found = true
			require.Equal(t, severityPtr(protocol.DiagnosticSeverityError), diag.Severity)
			break
		}
	}
	require.True(t, found, "undefined-dependency diagnostic is reported")
}

func TestDocument_DiagnosticsEmptyForCleanFile(t *testing.T) {
	src := `
VERSION = "1.0.0"

task build {
    echo $VERSION
}
`
	d := parse(testURI, src, 1)
	require.Empty(t, d.diagnostics())
}

func TestDocument_DocumentSymbolsMirrorTopLevelDeclarations(t *testing.T) {
	src := `
VERSION = "1.0.0"

task build {
    echo building
}

namespace db {
    task migrate {
        echo migrating
    }
}
`
	d := parse(testURI, src, 1)

	symbols := d.documentSymbols()
	require.Len(t, symbols, 3)

	// Grouped by declaration kind: tasks first, then variables, then
	// namespaces. This matches QuakeFile field order, not the order
	// declarations appear in the source file.
	require.Equal(t, "build", symbols[0].Name)
	require.Equal(t, protocol.SymbolKindFunction, symbols[0].Kind)

	require.Equal(t, "VERSION", symbols[1].Name)
	require.Equal(t, protocol.SymbolKindVariable, symbols[1].Kind)

	require.Equal(t, "db", symbols[2].Name)
	require.Equal(t, protocol.SymbolKindNamespace, symbols[2].Kind)
	require.Len(t, symbols[2].Children, 1)
	require.Equal(t, "migrate", symbols[2].Children[0].Name)
}

func TestDocument_DefinitionResolvesTaskDependency(t *testing.T) {
	src := "task build {\n    echo hi\n}\n\ntask ship => build {\n    echo shipping\n}\n"
	d := parse(testURI, src, 1)

	// Point at "build" in the dependency list of `ship`.
	depOffset := indexOf(src, "=> build") + len("=> ")
	require.Equal(t, 'b', rune(src[depOffset]))

	pos := posAt(src, depOffset)
	loc := d.definition(pos)
	require.NotNil(t, loc, "definition should resolve to task build")
	require.Equal(t, testURI, loc.URI)

	// Definition should point to the line of `task build`.
	require.Equal(t, protocol.UInteger(0), loc.Range.Start.Line)
}

func TestDocument_DefinitionResolvesVariable(t *testing.T) {
	src := "VERSION = \"1.0\"\n\ntask show {\n    echo $VERSION\n}\n"
	d := parse(testURI, src, 1)

	// Point at "VERSION" after the `$`.
	off := indexOf(src, "echo $VERSION") + len("echo $")
	require.Equal(t, 'V', rune(src[off]))

	pos := posAt(src, off)
	loc := d.definition(pos)
	require.NotNil(t, loc)
	require.Equal(t, protocol.UInteger(0), loc.Range.Start.Line, "definition jumps to the top-level VERSION assignment")
}

func TestDocument_DefinitionNilForUnknownSymbol(t *testing.T) {
	src := "task build {\n    echo hi\n}\n"
	d := parse(testURI, src, 1)

	// Cursor inside "echo" — not a defined task or variable.
	off := indexOf(src, "echo")
	pos := posAt(src, off)
	require.Nil(t, d.definition(pos))
}

// posAt converts a byte offset in text to an LSP Position for test
// readability. Production code goes the other way.
func posAt(text string, offset int) protocol.Position {
	li := newLineIndex(text)
	return li.position(offset)
}

// indexOf is strings.Index wrapped with a panic on not-found, so
// test setup fails loudly rather than silently passing -1 into the
// subsequent assertions.
func indexOf(haystack, needle string) int {
	i := strings.Index(haystack, needle)
	if i < 0 {
		panic("indexOf: needle not found: " + needle)
	}
	return i
}
