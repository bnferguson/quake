# Quake Review Guide

Context for reviewing code in this repo. Lives on `lsp/meta` only.
Load this before reviewing Quake changes — it captures idioms the
existing code already follows that a generic Go reviewer would miss.

## Architecture in one paragraph

`parser/` turns Quakefile source into an AST; every node carries a
`Position` so editor tooling can map AST → source. `evaluator/`
walks the AST to run tasks. `workspace/` discovers and merges all
the files that contribute to a project (main `Quakefile` + auxiliary
`*.quake` files in `qtasks/` + discovered Go tasks). `internal/gotasks/`
finds exported functions in `qtasks/*.go` files and generates a tiny
dispatcher `main.go` it can `go run`. `main.go` is the CLI front end:
flags, subcommand dispatch, and a couple of Claude-backed authoring
helpers (`-g`, `--init`).

```
parser    →  AST with Positions
evaluator →  executes AST (reads parser types)
workspace →  loads + merges per-project files (reads parser; uses gotasks)
internal/gotasks  →  Go-function-as-task discovery + codegen
main.go   →  CLI
```

Two LSP-adjacent branches also exist:

- `lsp/meta` — planning / this doc, never merges.
- `lsp/01` through `lsp/04` — stacked PRs building an LSP. See
  `LSP_PLAN.md` for the roadmap and branch status.

## Dependencies worth knowing

- **`github.com/lab47/peggysue`** — parser combinators. Grammar rules
  are fields on a `Grammar` struct built in `NewGrammar().init()`.
  Peggysue calls `SetPosition` on any AST node that implements
  `SetPositioner` — that's why every AST type has `Position` and
  `SetPosition`.
- **`miren.dev/mflags`** — CLI flags. `flags.BoolVar`, `StringVar`
  with both long and short forms. Returns `mflags.ErrHelp` when the
  user asked for help.
- **`github.com/stretchr/testify/require`** — tests use `require`,
  not `assert`. Short-circuit on first failure is the norm.
- Go `1.24`. Use `t.TempDir()`, `t.Chdir()`, `t.Cleanup()`.

## Error handling idioms

**Wrapping.** `fmt.Errorf("verb %s: %w", path, err)` — lowercase verb,
no "failed to", include the path. Reviewers should reject "failed to
read Quakefile" and similar; match `os.PathError` style.

**Parser return shape.** `parser.ParseQuakefileWithSource` returns
`(QuakeFile, bool, error)`. Check `err` first, then `!ok`. Do NOT
collapse to `if !ok || err != nil` with a single `%w(err)` — that
silently wraps `nil` when `ok == false, err == nil`:

```go
// Correct
if err != nil { return fmt.Errorf("parse %s: %w", path, err) }
if !ok        { return fmt.Errorf("parse %s: unknown parse failure", path) }
```

**Sentinel errors.** `workspace.ErrQuakefileNotFound` is the model
— an `errors.New` at package scope, documented, `errors.Is`-checkable.
Reach for a sentinel whenever a caller (especially the future LSP)
will need to branch on the error category.

**Non-fatal problems.** Don't `fmt.Fprintf(os.Stderr, ...)` from
library code. Collect warnings as `[]error` on the result struct
(see `Workspace.Warnings`) and let the CLI layer (or the LSP) decide
what to do with them. The CLI's `printWarnings` helper is the only
place warnings become stderr output.

## AST design

Rules every AST node follows — add a new one only if you follow all
of them:

1. Exported struct with JSON tags on every field.
2. A `Position Position` field tagged `json:"-"` (positions do not
   round-trip through JSON).
3. A `SetPosition(start, end, line int, filename string)` method.
   Peggysue discovers this via the `SetPositioner` interface.
4. Interface membership (e.g. `Expression`, `CommandElement`) via a
   marker method: `func (Foo) expression() {}` — value receiver, empty
   body.
5. Custom `MarshalJSON`/`UnmarshalJSON` only when the struct holds an
   interface-typed field that needs a type tag on the wire (see
   `Command`, `Variable`).

**Pointer vs value dance.** Parser actions return `*T` so peggysue
can mutate `Position` in place; the public AST exposes value-typed
`T` in struct fields. `cmdElemValue`, `exprValue`, etc. dereference
at the boundary. Don't invert this — peggysue needs the pointer, and
callers benefit from value semantics on the struct.

**Command-element positions are zeroed.** `parseCommands` re-parses
each command line in isolation, so peggysue would record offsets
into the sub-buffer rather than the source file. Until command
parsing is inlined into the main grammar, `StringElement`,
`VariableElement`, `ExpressionElement`, and `BacktickElement`
positions are intentionally `Position{}`. Tests assert this. Don't
"fix" it without also inlining the grammar.

## Peggysue patterns

- Grammar rules live as `p.Rule` fields on `Grammar`. `NewGrammar()`
  returns the built grammar; `(*Grammar).init()` wires every rule.
- Forward-referenced rules use `p.R("name")` and later assignment.
- Rule actions use `p.Action`, `p.Transform`, `p.Named`, and the
  `p.Values` helper with `.Get(name).(T)` type assertions.
- When you need an action to return a positioned AST node that might
  be wrapped by an outer rule, use the `positionGuard` pattern from
  `parser.go` — an unpositioned wrapper that hides an already-set
  child Position from outer dispatch.

## Workspace conventions

- `Load(ctx, mainPath)` — takes `context.Context`. Check `ctx.Err()`
  between file reads and before expensive steps (Go task discovery).
- `Workspace` is **not** safe for concurrent use. LSP-style reload
  should build a fresh `Workspace` and swap it in atomically, not
  mutate in place.
- `Sources` lists every `.quake` / `Quakefile` path that contributed.
  Go task source files are not in `Sources` (they're tracked per-task
  via `Task.SourceFile`).
- `Close()` is idempotent; it releases compiled Go-task dispatchers.
- `Warnings` is a public `[]error` field on the struct, by design.
  Don't wrap it in a method unless you have a reason — Go convention
  is public fields on result structs.

## Testing conventions

- Files colocated: `foo.go` ↔ `foo_test.go`, same package (white-box).
- `testify/require` everywhere. `require.Equal(t, expected, got)` —
  expected first. No `assert`.
- Test names: `TestSubject_SpecificCase`. Underscore separates the
  subject from the scenario.
- Regression tests for fixed bugs are `TestBug_Description` and carry
  a doc comment explaining the bug they guard against.
- When a test compares against a full expected AST literal, use the
  `parseNoPos` helper (or `stripPositions`) so the expected tree
  doesn't need to know byte offsets. When a test is specifically
  about positions, call `ParseQuakefile` directly.
- Use `t.TempDir()` for filesystem state. Use `t.Chdir()` when a test
  needs to verify CWD-sensitive behavior (e.g. `FindQuakefile`
  walking up from the CWD).
- `t.Cleanup(func() { require.NoError(t, ws.Close()) })` — assert on
  the close error. Silently discarding it with `_` is the kind of
  thing `errcheck` would flag.

## Naming

- Exported: `MixedCaps`. Unexported: `mixedCaps`. No underscores in
  identifiers — they're reserved for test names.
- Package names are short, lowercase, no underscores: `parser`,
  `evaluator`, `workspace`, `gotasks`, `color`.
- `internal/` holds packages that aren't part of the public API.
  `internal/gotasks`, `internal/color`. Reach for `internal/` whenever
  a package doesn't need to be imported outside the module.
- Avoid stuttering. `workspace.Workspace` is acceptable (the type is
  the package's raison d'être). `workspace.FindQuakefile` is fine.
  `workspace.WorkspaceLoader` would not be.

## Comment conventions

- Every exported symbol has a doc comment starting with the symbol
  name: `// Foo does X.`, not `// Does X.`.
- Multi-paragraph doc comments on types when there's genuine nuance
  (see `Position` in `parser/ast.go`). Keep them, they earn their
  space.
- Inline comments explain **why**, not **what**. Good example from
  `parser/parser.go`:

  > positionGuard wraps an already-positioned AST pointer so the
  > containing Action's SetPositioner dispatch can't overwrite it.

  That's describing a non-obvious interaction, not restating the
  code. Reviewers: push back on narrating-what-the-code-does comments.

## main.go

- `main` immediately calls `realMain` which returns an exit code.
  Makes integration tests possible without `os.Exit`.
- Flag parsing via `mflags`; `errors.Is(err, mflags.ErrHelp)` is the
  convention for "user asked for help".
- `--` separates multiple task invocations: `quake build -- test`.
  Keep this if you're touching argument parsing.
- Claude-backed helpers (`generateTaskWithClaude`, `initQuakefileWithClaude`)
  are self-contained and longer than the rest of the file. They're
  largely out of scope for most reviews — flag only if a change
  actively touches them.

## What lives where — flag misplacement

When reviewing a PR, check that each change landed in the right layer:

- File I/O that's specific to a single project: `workspace/`, not
  `main.go`.
- AST traversal (finding symbols, building reverse indices, detecting
  cycles): belongs in a future `analysis/` package per `LSP_PLAN.md`,
  not in `parser/` or `evaluator/`.
- Editor/LSP protocol concerns: `internal/lsp/` (planned). Do not
  let LSP concepts leak into `parser/`, `evaluator/`, `workspace/`,
  or `analysis/`.
- CLI output formatting: `main.go`. Library code should return data,
  not print.

## Things deliberately not done here

Push back on PRs that introduce these without a clear reason:

- No logging framework. `fmt.Fprintf(os.Stderr, ...)` for CLI-visible
  warnings; nothing for library code.
- No `context.Context` threaded through the evaluator. Task execution
  is synchronous and cancellable via signal, not `ctx`.
- No concurrency in `parser/`, `evaluator/`, or `workspace/`. The
  LSP layer (when it lands) owns its own sync model.
- No external AST transformation API. The AST is an implementation
  detail; changes to shape don't trigger semver decisions.

## Quick reviewer checklist

Before approving a PR that touches this codebase:

1. **Errors.** Lowercase, wrapped with `%w`, include the path. No
   `fmt.Fprintf(os.Stderr, ...)` from library code.
2. **New AST node?** `Position`, `SetPosition`, JSON tags, marker
   method if it satisfies an interface. Tests cover positions.
3. **New exported symbol?** Has a doc comment starting with the name.
4. **Tests.** `testify/require`, `TestSubject_Case` naming, `t.TempDir`
   for filesystem work, asserts on `Close()` errors.
5. **Layer check.** Did I/O-ish code land in `workspace/`? Did AST
   analysis land in `analysis/`? Did formatting stay in `main.go`?
6. **LSP trajectory.** If this change touches `parser/`, `workspace/`,
   or anything that will be consumed by the LSP, does it preserve
   per-file position info and sentinel-error semantics?
