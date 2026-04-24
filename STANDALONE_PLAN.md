# Quakefile LSP — Standalone Spinout Plan

Working doc for moving the Quakefile language server out of the
`bnferguson/quake` fork into its own repository. Lives on the
`lsp/meta` branch alongside [`LSP_PLAN.md`](LSP_PLAN.md); never
merges into `main` or upstream.

## Why

The LSP work adds ~4,600 lines to `miren/quake` (near-doubling the
codebase). Most of that is the server itself — a feature upstream
may reasonably decide not to own. A standalone repo keeps miren's
project lean, lets LSP iteration stop waiting on upstream review
cycles, and sets a clear semver boundary on what the LSP depends on.

The cost is parser-version skew: the standalone pins a snapshot and
lags Quakefile grammar changes until someone bumps it. Accepting
that cost is the central tradeoff.

## Strategy

Four phases. The first gets us off the fork; the others reduce how
much we carry privately as upstream absorbs the infrastructure bits.

### Phase 1 — spin out, import positions from the fork

Goal: ship a standalone `quake-lsp` binary, importing parser from
the fork's `lsp/01-ast-positions` branch because positions aren't
upstream yet.

New repo layout (tentative, module path TBD):

```
quake-lsp/
├── cmd/quake-lsp/main.go      # thin entry point — calls lsp.Serve
├── analysis/                  # copied from fork lsp/03
├── workspace/                 # copied from fork lsp/02
├── internal/lsp/              # copied from fork lsp/04..08
├── go.mod
└── README.md
```

`parser/` is an import, not a copy:

```go
// quake-lsp/go.mod
module github.com/bnferguson/quake-lsp    // TBD

go 1.24

require miren.dev/quake v0.0.0-<date>-<sha>

// Points the import at my fork until lsp/01 lands upstream. Pin to
// a SHA, not a branch name — Go's module resolver needs immutable
// refs.
replace miren.dev/quake => github.com/bnferguson/quake <lsp/01 commit sha>
```

Carried code stays on its existing package paths (`analysis`,
`workspace`) so the call sites in `internal/lsp` don't change on
spinout. When the packages come back as upstream imports in later
phases, we drop the copies without moving use sites.

Bootstrap checklist:

- [ ] Pick repo name + module path (section below)
- [ ] Create repo, copy over `analysis/`, `workspace/`, `internal/lsp/`
- [ ] Add `cmd/quake-lsp/main.go` (5–10 lines, calls `lsp.Serve`)
- [ ] Configure `go.mod` with the `replace` directive
- [ ] Port the fork's CI (tests + `go vet`)
- [ ] Point `zed-quakefile` at the new binary name
- [ ] Retire the `quake lsp` subcommand in the fork (or leave it
      referencing the new module as a convenience — TBD)

### Phase 2 — pitch `lsp/01-ast-positions` upstream

Once the standalone is working off the forked parser, open a PR
against `mirendev/quake:main` with just `lsp/01-ast-positions`.
It's the smallest, clearest additive change: adds a `Position`
field + `SetPosition` method on every AST node and wires up the
`peggysue.SetPositioner` interface. Useful to any future editor
tooling, not just our LSP.

If it lands: update the standalone's `go.mod` to import
`miren.dev/quake` directly and drop the `replace` line.

If it doesn't: stay on the fork. Rebase `lsp/01-ast-positions` onto
miren's `main` whenever the parser changes; bump the replace SHA.
Cheap ongoing cost — the branch is 438 lines, mostly test data.

### Phase 3 — pitch `lsp/03-analysis-package` (with `lsp/02` as prereq)

After positions land, pitch analysis. The upstream-facing argument
is a `quake check` subcommand: parse the Quakefile, run the
diagnostics (undefined deps, dependency cycles, unresolved
variables), exit non-zero on problems. CI-friendly, catches typos
at author-time, doesn't require `quake build` to surface a broken
dep list.

`lsp/02-workspace-package` is the dependency: to check across
auxiliary `*_Quakefile` files and `qtasks/` discovery, `quake
check` needs the same loader the LSP uses. Easier to pitch as one
bundle — "here's `quake check` and the refactor that makes it
possible" — than as two separate refactor PRs with no user-facing
payoff.

If both land: standalone drops its copied `analysis/` and
`workspace/`, imports them from miren. If miren accepts 01 but not
03, standalone keeps `analysis/` (and `workspace/`) forever; that's
still a smaller carry than the whole LSP.

### Phase 4 — steady state

Best case, after phases 2 and 3 complete:

```go
// quake-lsp/go.mod
require miren.dev/quake v<upstream-tag>
```

And the standalone repo shrinks to:

```
quake-lsp/
├── cmd/quake-lsp/main.go
├── internal/lsp/       # the only substantial code
└── go.mod
```

All the substantive dependencies are public imports. Grammar changes
in miren's parser still need a dep bump, but the skew risk is
narrowed to parser-shape changes — not every refactor in between.

## Repo structure (Phase 1 target)

What moves, what imports, what stays:

| From fork branch | To standalone repo | How |
|---|---|---|
| `parser/` (lsp/01) | — | imported via `require` + `replace` |
| `workspace/` (lsp/02) | `workspace/` | copy |
| `analysis/` (lsp/03) | `analysis/` | copy |
| `internal/lsp/` (lsp/04..08) | `internal/lsp/` | copy |
| `main.go` | `cmd/quake-lsp/main.go` | rewrite (thin) |
| `evaluator/`, `internal/gotasks/`, etc. | — | not needed by LSP |

Import paths inside `internal/lsp/` change from `miren.dev/quake/analysis`
and `miren.dev/quake/workspace` to `<standalone-module>/analysis` and
`<standalone-module>/workspace`. Trivial mechanical find-and-replace.
When phases 2+3 land upstream, we revert those imports back to
`miren.dev/quake/...` and delete the copies.

## Upstreaming fallback matrix

What happens if each pitch is declined:

| Scenario | Consequence for standalone |
|---|---|
| 01 lands, 02+03 land | Phase 4 steady state. Minimal carry. |
| 01 lands, 02+03 declined | Standalone owns `workspace/` + `analysis/`. Bump parser via upstream tags. Most of the ongoing cost is in `workspace/` since it tracks `main.go` loading changes. |
| 01 declined | Carry `parser/` patch on fork branch indefinitely. Rebase when grammar changes. Still fine — the diff is small and test-heavy. |

The bet: 01 is hard to decline — it's additive, useful beyond LSP,
and 200 lines of actual source + 230 lines of tests. 02/03 are a
real product pitch riding on "ship `quake check`", so they stand or
fall on whether miren wants that subcommand.

## ~~Open questions~~ (Answered)

- ~~**Repo name / module path.** `quake-lsp`, `quakelsp`, `quake-language-server`?
  Vanity domain or plain `github.com/bnferguson/...`? The name
  affects the binary (`quake-lsp` vs `quakels`) — Zed's extension
  config is the first downstream consumer to update.~~
  - `quake-lsp`
- ~~**Release cadence.** Tags from day one, or `go install` from
  main until there's a downstream pinning need? Homebrew tap
  (pair with `zed-quakefile`)?~~
  - go install from main for now.
- ~~**Backfill `quake lsp` subcommand.** Leave it in the fork as a
  redirect ("install quake-lsp separately"), remove it, or keep it
  working by `go install`-ing the standalone from the fork's build?~~
  - Leave it. The subcommand lives on fork branches `lsp/04`–`lsp/08`
    and keeps working for anyone already installed from there; the
    standalone just becomes the recommended install going forward.
    Phase 1 only touches `lsp/01` as an import, so there's nothing to
    reconcile today.
- ~~**Config surface.** The LSP reads nothing from the workspace
  today. If a `.quake-lsp.toml` ever shows up, it ships with the
  standalone — not upstream's problem.~~
  - yeah we don't need anything here right now

## How this doc stays current

When phase 1 starts, add a live status table here (similar to
`LSP_PLAN.md`'s branch/PR mapping). Link the standalone repo once
created. Update the upstreaming fallback matrix with real outcomes
as PRs land or get declined.
