# Quakefile LSP — Plan

Working doc for the Quakefile language server effort. Lives on the `lsp/meta` branch only — never merges into `main` or upstream PRs. Update it whenever a branch lands, gets rebased, or shifts scope.

## Goal

First-class editor support for Quakefiles: go-to-definition, find-references, rename, autocomplete, diagnostics. Shipped as a `quake lsp` subcommand, reusing the existing parser + evaluator. Zed (via [zed-quakefile](https://github.com/bnferguson/zed-quakefile)) will be the first consumer.

## Strategy

- Break the work into four stacked branches, each one a self-contained PR against miren/quake.
- The first three are just useful infrastructure (positions in AST, workspace package, analysis package) — miren may want them even if they never ship an LSP.
- The fourth is the LSP itself. If miren doesn't want it upstream, it lives in `bnferguson/quake` forever.
- Each branch builds on the previous. Reviewable in isolation; shippable incrementally.

## Architecture (target)

```
quake/
├── parser/                     # + Position struct + SetPosition methods
├── evaluator/                  # unchanged
├── workspace/                  # NEW — extracted from main.go
│   ├── workspace.go            # Workspace type, merged AST
│   └── discover.go             # findQuakefile, *_Quakefile resolution
├── analysis/                   # NEW — semantic analysis over AST
│   ├── symbols.go              # SymbolTable: task name → definition
│   ├── references.go           # reverse index: task → callers
│   ├── diagnostics.go          # undefined deps, cycles, unresolved refs
│   └── resolve.go              # namespaced name lookup
├── internal/lsp/               # NEW — LSP protocol layer (private)
│   ├── server.go               # glsp handlers
│   ├── document.go             # per-document cache
│   └── conversions.go          # AST Position ↔ LSP Range
└── main.go                     # + `quake lsp` subcommand
```

`workspace/` and `analysis/` are public packages — the CLI uses them too (better errors, `quake check`). `internal/lsp/` is private; the protocol layer has no reusable surface.

## LSP protocol surface

### Phase 1 — MVP (ships in `lsp/04-lsp-server`)

| Method | Use |
|---|---|
| `initialize` / `initialized` / `shutdown` / `exit` | Lifecycle |
| `textDocument/didOpen` / `didChange` / `didSave` / `didClose` | Full-sync |
| `textDocument/publishDiagnostics` | Parse errors, undefined deps, cycles, unresolved `$VAR` refs |
| `textDocument/documentSymbol` | Outline: tasks, namespaces, variables |
| `textDocument/definition` | Jump from `=> build` → `task build`, `$VAR` → assignment |

### Phase 2 — navigation depth

| Method | Use |
|---|---|
| `textDocument/references` | Who depends on task X |
| `textDocument/hover` | Task description + args + deps; variable value |
| `textDocument/documentHighlight` | Same-symbol highlighting in current file |

### Phase 3 — authoring help

| Method | Use |
|---|---|
| `textDocument/completion` | Task names in `=>`, variable names after `$`, identifiers in `{{...}}` |
| `textDocument/rename` | Rename task + update every reference workspace-wide |
| `workspace/symbol` | Cmd-T across workspace |

### Phase 4 — polish (optional)

| Method | Use |
|---|---|
| `textDocument/codeAction` | "Extract commands into new task" |
| `textDocument/formatting` | If we agree on a canonical style |

### Explicitly skipped

`signatureHelp`, `typeDefinition`, `implementation`, `foldingRange`, `selectionRange`, `semanticTokens` — tree-sitter already covers the ones editors need.

## Branch → PR mapping

| # | Branch | Base | PR target | Status | Notes |
|---|---|---|---|---|---|
| 0 | `lsp/meta` | `main` | never | 🟠 in progress | This doc lives here; never goes upstream |
| 1 | `lsp/01-ast-positions` | `main` | `miren/main` | 🟡 complete | Add `Position` to AST nodes; implement `peggysue.SetPositioner` |
| 2 | `lsp/02-workspace-package` | `lsp/01` | `miren/main` (after 1 lands) | 🔴 not started | Extract `loadAllQuakefiles` + friends from `main.go` into `workspace/` |
| 3 | `lsp/03-analysis-package` | `lsp/02` | `miren/main` (after 2 lands) | 🔴 not started | `analysis/` package: symbols, refs, diagnostics |
| 4 | `lsp/04-lsp-server` | `lsp/03` | `miren/main` (optional) or stays in fork | 🔴 not started | `quake lsp` subcommand, Phase 1 LSP methods |

Status legend: 🔴 not started · 🟠 in progress · 🟡 complete locally (no PR yet) · 🟢 PR open · ✅ merged upstream · ⚪ blocked

Upstream PR flow: push branch, open PR with base `miren/quake:main` (or the prior stacked branch if reviewed together). Plan file is never included.

## Decisions

- **Subcommand over separate binary.** `quake lsp` on stdin/stdout. Parser version = LSP version, no skew.
- **glsp for LSP plumbing.** `github.com/tliron/glsp` — covers JSON-RPC + protocol types. Full-sync initially; no incremental.
- **Full reparse per change.** Peggysue doesn't do incremental; Quakefiles are small enough it doesn't matter.
- **Workspace-wide symbols.** References/rename cross `*_Quakefile` boundaries.

## Open questions

- **Upstream path for Phase 4 LSP server.** Pitch as an issue first or send the PR? Probably pitch — adding an LSP commits miren to editor-integration support forever.
- **Multi-root workspaces.** Zed can open two unrelated projects in sub-folders. Start single-root; grow into multi-root if it actually comes up.
- **Unresolved-variable diagnostics for `{{env.X}}`.** Env vars are runtime values. Warn on unresolved non-env expression refs only, or skip `env` entirely? Leaning skip.

## How to work the stack

Rebase discipline is everything. When `lsp/01` changes, rebase the rest:

```
git checkout lsp/02-workspace-package
git rebase lsp/01-ast-positions

git checkout lsp/03-analysis-package
git rebase lsp/02-workspace-package

# ... and so on
```

When the first branch lands upstream and `main` advances:

```
git checkout main
git pull upstream main
git checkout lsp/02-workspace-package
git rebase --onto main <old-lsp-01-tip>     # drops the obsolete lsp/01 commits
# then propagate the rebase down the stack
```

Update this plan's "Status" column each time a branch moves.
