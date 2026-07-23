# Agent Notes (GoToTS)

`AGENTS.md` and `CLAUDE.md` must remain byte-identical. Apply every change to
both and verify with `cmp`.

## Begin With WCBUBWHB

Every task, review, design, code change, test, generated-surface change, and
performance change begins with explicit WCBUBWHB analysis. Read-only discovery
may precede it only to establish facts.

Answer before editing:

1. What is the observed source and artifact problem?
2. What broader semantic class and sibling paths exist?
3. Which layer owns the truth?
4. What is the highest layer that eliminates the class?
5. Which duplicate, fallback, workaround, or alternate route must be deleted?
6. What is the simplest exact result without preserving current shape?
7. What are the staticness, size, typecheck, runtime, extension, and consumer
   consequences?
8. What Go input -> semantic decision -> generated TypeScript examples prove
   the shape?
9. What independent differential/contract and mutation test proves it?
10. What broad search and deletion checks prove no sibling path remains?

Do not patch a reproduction and justify it afterward. A second instance of the
same workaround stops feature work and reopens the semantic owner.

## Governing Product

GoToTS is a general Go-to-TypeScript compiler. `docs/spec/` is authoritative.
No acceptance corpus, including `typescript-go`, may become a production
dependency, semantic condition, package-name rule, or privileged profile.

Every selected Go construct has one context-aware typed disposition. Unknown
or unclassified forms fail before emission. Source-available dependencies are
ordinary Go source. Runtime, manually completed `gostdlib`, true external
contracts, manual units, and customer extensions have the separate owners
defined by the specification.

## Architecture Invariants

- One semantic fact has one authoritative producer and typed identity.
- Parent visitors assign grammatical roles; children do not inspect parents or
  source text to infer meaning.
- The semantic model is target-independent.
- Whole-program facts seal before one total immutable plan is built.
- Semantic reachability seals before planning and excludes unreachable
  definitions from planning/lowering. Post-completion implementation
  reachability is a separate graph; neither substitutes for the other.
- Lowering consumes semantic operations plus their plans and makes no semantic
  or representation decisions.
- Every selected source file or synthetic semantic owner has one `OwnerRegion`.
  Every grammatical `OccurrenceID` has one canonical immutable occurrence
  payload. Regions, headers, boundaries, containment paths, and executable
  regions reference that payload; they never copy kind, parent, edge, role,
  order, token, or span facts into a second record.
  Every implementation-bearing Go construct has one `DefinitionID`, exactly one
  `DefinitionSite` referencing a complete path in the normalized sparse
  containment graph, exactly one `HeaderRegion`, exactly one
  `ExecutionBoundary`, and a separate `DefinitionSelection`. An
  `ExecutableRegion` exists exactly for every full-semantic definition. The
  owner/containment/definition/site/header/boundary facts are depth-independent
  and owned once; later semantic/call/dispatch/reachability references may
  repeat.
  An excluded executable interior is unreachable through the finalized API; a
  raw parent node plus boundaries, traversal flag, or consumer skipping is
  forbidden.
- One implementation path exists. No fallback, old/new flag, retry with
  changed semantics, compatibility reader, or parallel state survives.
- No semantic recovery through `any`, `unknown`, unchecked casts, reflection,
  spelling lookup, text scans, or host-shape probing.
- Ordinary calls preserve source argument shape. Hidden operation protocols
  require local typed necessity and are never the default.
- Interface dispatch cannot grow with implementer count at each call.
- Generated/manual ownership is per declaration or body, never per file.
- Regeneration starts from an empty generated baseline.
- Reachability traverses generated, manual, runtime, stdlib, external,
  extension, initialization, callback, generic, and dynamic-dispatch edges.
- Extensions consume finalized selected evidence and never re-enter analysis.

## Spec-Adequacy Gate

No phase or shared abstraction enters implementation until its governing
specification defines its closed input classes, authoritative owner, exact
output schema, lifecycle and mutability boundary, context/containment rules,
conservation law, downstream contract, superseded-path deletion list, positive
and adversarial source examples, independent verifier and mutations, and
size/memory/time bounds. Terms such as "unit," "boundary," "exact,"
"immutable," and "independent" are not acceptance criteria unless their
observable representation and failure cases are defined.

The gate challenges the words with degenerate implementations. A discriminator
enum is not its payload; a span is not topology; a hash is not authority; counts
are not identity joins; a synthetic foil is not a mutation of production work;
and one increment per visitor does not measure work hidden inside that visit.
If the trivial substitute can satisfy the prose, the specification is
inadequate and must be replaced before code begins.

## No-Compromise Design Review

Before changing governing authority or activating a phase, perform a separate
design-only WCBUBWHB review. The design may proceed only when there is no known
architectural compromise against correctness, genericity, scalability,
staticness, source shape, ownership, lifecycle, verification, or bounded cost.
Schedule pressure, current code shape, and edit size are not design inputs.

The review must:

1. decompose the domain into orthogonal facts and forbid one field, identity,
   enum, digest, or switch from standing for two facts;
2. give every relation an exact cardinality and distinguish containment,
   semantic use, execution order, ownership, and reachability;
3. exercise the cross-product of construct kinds, evidence depths, local versus
   certified sources, explicit versus implicit forms, parent/child selections,
   and transformed/checked views;
4. prove that a future construct extends one closed algebra/catalog rather than
   adding a parallel case table;
5. define independent derivation and mutations against the real production
   owner, not shared helpers or synthetic demonstrations;
6. state asymptotic work/storage and all source-size, typecheck, memory, and
   runtime consequences before implementation;
7. identify every existing schema/producer/consumer/test/comment that the clean
   design deletes; and
8. reread literal compliance adversarially and reject any wording that permits
   an empty payload, duplicate owner, depth-selected topology, or unmeasured
   work.

Accepted specification changes replace ambiguous or contradictory text; they
do not append a second interpretation. Implementation reports must re-run this
review against actual artifacts before phase exit. A newly discovered ambiguity
reopens authority first, not a local patch.

When a review exposes inadequate governing design, freeze dependent
implementation, correct `docs/spec/` atomically, perform an independent
specification-adequacy pass, and only then issue one replacement directive bound
to that reviewed authority revision. Never ask an implementer to infer missing
architecture from review prose, continue code while authority is changing, or
implement a “temporary” bridge to the intended design.

## Product Quality Is Correctness

Semantic equivalence, strict TypeScript, source-shaped architecture, generated
size, typecheck time/RSS, generation cost, and runtime cost are simultaneous
requirements. A change is incomplete when one is unmeasured or knowingly
blocked.

Inspect actual artifacts. Every technical example shows Go source first, the
semantic decision, generated TypeScript, why a simpler form is or is not exact,
cost impact, and independent proof. Review calibration fixtures, ordinary
samples, and the twenty largest/highest-expansion changed artifacts.

Stop immediately when definitions repeat, identity is overwritten, interface
dispatch expands per implementer, ordinary calls gain hidden plumbing, erased
recovery appears, bytes become unattributed, or size/typecheck/runtime grows
without typed necessity. Do not raise thresholds, add allowlists, suppress
diagnostics, or special-case the largest corpus site.

## Manual And Environment Work

Generated and manual declarations may coexist in one file. Generated bodies
carry post-format hashes; a mismatch or absent generated header means manual.
Writing TypeScript is sufficient—never require user-authored attachment JSON.

All automatic output is recreated during regeneration. Compatible manual code
is structurally overlaid, fully type-resolved, and included in the complete
graph. Unreachable manual code is reported, preserved in the editable workspace
by default, and removed only through explicit dry-run/apply pruning.

`gostdlib` declarations come from the selected `GOROOT`; behavior is manually
completed through the same placeholder/hash mechanism. True externals receive
exact typed contracts and placeholders. Reachable unresolved placeholders
block publication.

The existing customer extension contract must remain compatible through one
generic finalized-evidence boundary. No textual patching or checker re-entry.

## Code Discipline

- Build the cleanroom architecture from the specification. Reuse an existing
  component only after its cleanroom admission review; never bulk-copy a
  subsystem or preserve it behind a facade.
- Use validating constructors and closed enums. Errors are typed; error strings
  never select behavior.
- Parent-directed construct classification carries every semantic context fact
  required by the catalog, including declaration token/class. A child node kind
  alone may not decide whether it is an implementation definition.
- Do not create `v2`, `legacy`, `compat`, `fallback`, `util`, `utils`,
  `helper`, `helpers`, or `misc` packages/files.
- Keep maintained non-generated implementation files semantically focused and
  below 600 physical lines. The fixed six-file governing specification is
  bounded by responsibility rather than this code-file line limit.
- Use descriptive filenames, not numeric shards.
- Generate TypeScript only through the typed target AST and one formatter.
- Use `apply_patch` for file edits.
- Delete replaced fields, helpers, tests, comments, and routes in the same
  architectural change.

## Verification And Reports

Start with focused construct/contract/differential/mutation tests, then strict
TypeScript and broader suites. Only the coordinator runs heavy whole-product
jobs; never stack them. Use bounded concurrency, disk-backed staging, memory
preflight, and resumable breadcrumbs.

Reports name exact identities, denominators, processing stages, gate states,
and parent deltas. IR-admitted, emitted, typechecked, executed, and certified
are distinct. “Complete” requires the specification's exact endpoint:
`18 pass / 0 fail / 0 blocked` at the clean published revision.

Before completion, broad-search for source-project conditions, sibling paths,
duplicate owners, fallback identities, stale generated code, obsolete tests,
and broken documentation references. Mutation-prove every gate's claimed
failure class.

## Directive Review Gate

Every outward review, implementation directive, correction, checkpoint reply,
and handoff is itself an engineering artifact. Draft it first, then perform a
separate review pass before sending it. Do not reason about code and issue the
mandate in the same pass.

The pre-send review must:

1. Bind the message to the exact inspected revision, current governing spec,
   active phase, and already-accepted phase-exit criteria.
2. Reproduce the decisive evidence and distinguish observed facts from
   hypotheses, recommendations, and future obligations.
3. Classify every item as a current-phase blocker, bounded work inside the
   active phase, deferred later-phase obligation, or non-issue.
4. Check the complete semantic class and truth owner; do not turn a new
   reproduction into a new local task.
5. Diff the draft against the previous directive. If it changes accepted
   scope, name the new evidence, state exactly what is superseded, and preserve
   every unaffected acceptance criterion.
6. Ensure the implementer sees one active endpoint and one dependency order.
   A message may not simultaneously stop work on a foundation and activate
   work that depends on that foundation.
7. Specify architectural invariants and observable exit evidence. Prescribe a
   particular implementation only when the invariant makes alternatives
   invalid; otherwise leave implementation choice to WCBUBWHB analysis.
8. Review source-size, generated-size, memory, typecheck, runtime, and scope
   consequences of the requested work. A mandate that causes broad work must
   contain typed necessity and a cost bound before it is sent.
9. Reread the draft from the implementer's perspective and ask whether literal
   compliance could create work that a later phase would discard, duplicate a
   truth owner, or expand scope beyond the active phase.
10. Record the review result as pass or revise the draft. A failed review is
    not sent.

Current-phase blockers alone may change the active mandate. Later-phase
obligations remain in the governing roadmap and are referenced, not added to
the current checklist. If a finding invalidates a shared foundation, the
message defines only the atomic replacement, its proof, and the downstream
restart boundary. If findings are bounded, the message may include remaining
substantial outcomes in the same active phase; it must not mechanically append
two to four future outcomes merely to appear decisive.

A substantial outcome is an end-to-end capability, architectural replacement,
or phase exit with its owner, migrated consumers, superseded-path deletion,
artifact inspection, independent tests, mutation proof, and applicable gates.
A helper, local reproduction, isolated test, or diagnostic reduction is not an
outcome. Substantial progress is required within the frozen active phase, not
by importing work from later phases.

After a directive passes review, it is frozen. Implementers proceed through it
without requesting routine confirmation and stop only for new reproducible
evidence that invalidates the active architecture, an unavailable external
prerequisite, or an unresolved product decision. Any amendment must pass this
gate again and explicitly replace the smallest affected portion. Reviewers do
not reopen accepted work without new evidence.

## Outward Response Format

Print the complete, copy/paste-ready message to the implementation team in the
response itself. Do not substitute a reviewer summary, file path, or instruction
to read another artifact for the team message. If a separate note to the user is
necessary, place it first, then print a `Message to Team` header followed by the
full team message. When no separate note is needed, output only the team message.

## Repository Safety

- Never force-push or delete remote branches/tags.
- Never use `git stash`.
- Work on feature branches unless the user explicitly directs otherwise.
- Keep meaningful work committed and pushed when requested.
- Use `.temp/` for scratch and `.analysis/` for local evidence; do not commit
  either.
- Keep generated files reproducible from checked-in inputs.
