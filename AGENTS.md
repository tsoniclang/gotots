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
- Lowering consumes semantic operations plus their plans and makes no semantic
  or representation decisions.
- Definitions are owned once; references may repeat.
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
- Do not create `v2`, `legacy`, `compat`, `fallback`, `util`, `utils`,
  `helper`, `helpers`, or `misc` packages/files.
- Keep maintained files semantically focused and below 600 physical lines.
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

## Review Forward Motion

Every review, checkpoint assessment, implementation mandate, and handoff must
move the project by a substantial dependency-ordered unit rather than create a
fix-one-finding-and-return supervision loop.

The reviewer must:

1. List the current findings first, with evidence, severity, truth owner, and
   required correction.
2. State whether the correction set is **foundational/massive** or **bounded**.
   A finding is foundational/massive only when it invalidates a shared semantic
   owner, identity domain, schema, phase boundary, or active architecture such
   that downstream work would be unsound or likely discarded, or when a
   stop-the-line condition prevents safe continuation.
3. If the findings are bounded, include the current corrections and normally
   the next two to four substantial dependency-ordered outcomes in the same
   task list. If the project is close to completion, include every remaining
   outcome through final acceptance instead.
4. If the findings are foundational/massive, define the complete atomic
   replacement and its exit proof, plus the exact downstream restart boundary;
   do not pad the mandate with work that would build on the invalid foundation.

A substantial outcome is an end-to-end capability, architectural replacement,
phase exit, or product milestone with its owner, migrated consumers, superseded
path deletion, artifact inspection, independent tests, mutation proof, and
applicable gates. A file edit, helper, local reproduction, isolated test, or
single diagnostic reduction is not a substantial outcome.

Implementers proceed through the combined task list without asking for
confirmation between items. They stop and re-plan only when a genuinely new
foundational/massive issue, unavailable external prerequisite, or required user
product decision makes the remaining plan unsafe. A checkpoint that reports
only bounded current fixes while leaving the already-known next substantial
work for another approval cycle is incomplete.

Batching never weakens WCBUBWHB, atomic replacement, evidence, or stop-the-line
requirements. Outcomes must remain cohesive and dependency-ordered; unrelated
work must not be bundled merely to make a checkpoint appear larger.

## Repository Safety

- Never force-push or delete remote branches/tags.
- Never use `git stash`.
- Work on feature branches unless the user explicitly directs otherwise.
- Keep meaningful work committed and pushed when requested.
- Use `.temp/` for scratch and `.analysis/` for local evidence; do not commit
  either.
- Keep generated files reproducible from checked-in inputs.
