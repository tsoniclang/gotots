# Verification

## Proof Principle

No single test proves translation. Each construct family is accepted only when
source coverage, target structure, strict typing, behavior, and cost agree at
one exact revision.

Verification must be independent at the compared boundary. A checker using the
same handler or canonicalizer as the producer is a consistency check, not
independent proof.

## Baseline Gates

The repository always enforces:

1. `AGENTS.md` and `CLAUDE.md` are byte-identical;
2. the pinned TS-Go schema digest and generated bindings agree;
3. only allowed packages import `go/ast`, `go/types`, and target factories;
4. no production package named or acting as `ir`, `plan`, `lower`, `catalog`,
   `inventory`, `legacy`, `compat`, or `fallback` exists;
5. production emission contains no raw TypeScript fragments or alternate
   formatter;
6. dynamic import, `.call`, `.apply`, `.bind`, `any`/`unknown` recovery,
   reflection, spelling dispatch, and source-text semantic scans are absent;
7. maintained non-generated files stay below 600 physical lines.

## Construct Coverage

Coverage has two distinct proofs:

- a toolchain-derived catalog of Go AST forms/tokens/built-ins is reconciled
  against handler or explicit-unsupported dispositions; and
- every selected project occurrence is observed entering exactly one handler.

The catalog is test authority only. Compilation does not produce or consume a
per-program inventory.

Mutation tests remove a handler, skip a child edge, alter child order, erase a
context role, and route one form to two handlers. Each mutation must fail with
the exact source occurrence or catalog form.

The demand-driven scheduler is verified independently from final target
declarations and source bindings: every selected root and resolved emitted
reference reaches exactly one target declaration or explicit obligation.
Mutations omit an enqueue, duplicate an owner, break a cycle reservation, and
silently drop a function-value/interface/callback target.

## Target-AST Proof

For every generated file:

1. all nodes are constructed through generated pinned-schema bindings;
2. schema validation succeeds;
3. the sole formatter produces the file;
4. TS-Go reparses the formatted file;
5. a normalized schema-level comparison matches the constructed tree;
6. source mappings and declaration ownership reconcile;
7. strict TypeScript resolution/typechecking succeeds.

Mutations introduce an unknown node kind, wrong field shape, duplicate
declaration, raw fragment, second formatter, and post-format edit. The owning
gate must reject each.

## Semantic Proof

Every implemented semantic family includes:

- focused source fixtures for ordinary and context-sensitive forms;
- boundary/property cases;
- Go-versus-generated-TypeScript differential execution;
- interaction cases for evaluation order, aliasing, copies, nil, panic,
  generics, method selection, and environment boundaries where applicable;
- real-project samples unrelated to the discovering corpus; and
- production-path mutations that prove the gate detects the intended defect.

Examples include:

- one-result versus comma-ok map indexing;
- one RHS producing multiple assignment results;
- side effects in parallel assignment and short-circuit expressions;
- nil pointer receiver methods that do and do not dereference;
- embedded methods whose Go static selection differs from TypeScript virtual
  dispatch;
- interfaces with many implementers, proving constant-size call sites;
- nested function literals requiring branch-local versus file-level placement.

## Architecture And Cost Proof

Each checkpoint reports absolute values and parent deltas for:

- maintained source files/lines;
- emitted bytes, tokens, and TS-Go AST nodes;
- largest twenty emitted declarations, expressions, and files;
- definitions, helpers, imports, placeholders, and ownership collisions;
- strict typecheck wall time and peak RSS;
- generation wall time and peak RSS;
- representative runtime wall time and memory;
- dynamic/staticness violations.

Aggregate improvements cannot hide a worsening tail. A one-line Go interface
call expanding into hundreds of target branches is a failed architecture even
if median output is small.

Scaling fixtures distinguish source-proportional work from growth by unrelated
package, declaration, or implementer count. Raising a threshold, adding an
allowlist, or suppressing a diagnostic is not a fix.

## Per-Capability Closure

Before dependent work begins, a capability must have:

1. a reviewed governing rule and concrete Go/TypeScript examples;
2. one authoritative handler/owner;
3. positive, negative, interaction, and mutation tests;
4. TS-Go AST construction/reparse proof;
5. strict typechecking and differential execution;
6. artifact-tail inspection and applicable cost bounds;
7. broad searches proving no alternate route remains;
8. a clean pushed revision carrying all evidence.

Unsupported neighboring constructs may remain explicit failures. A capability
is not accepted if it silently falls back, emits a placeholder while claiming
translation, or defers its own proof to a later milestone.

## Heavy Runs

Heavy tests run one process group at a time with:

- bounded worker concurrency;
- a timeout;
- disk-backed output and logs;
- current-stage and process breadcrumbs;
- an OS-enforced memory ceiling; and
- preserved crash/OOM/timeout evidence.

Up to 4 GiB may be used when justified. WCBUBWHB concerns architectural
cleanliness; memory limits are execution safety, not a substitute for design.

## Publication

Publication requires all selected constructs and reachable environment
obligations to be implemented, every generated module strict-typechecked, all
required differential/project suites green, no reachable placeholder, exact
deterministic regeneration, and all architecture/size/typecheck/runtime gates
passing at one clean revision.
