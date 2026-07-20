# Gotots Agent Instructions

`AGENTS.md` and `CLAUDE.md` must remain byte-identical. Any change to one must
be applied identically to the other and verified with `cmp`.

## Begin With WCBUBWHB

Every task, review, investigation, design, code change, test change, and
performance change begins with an explicit WCBUBWHB analysis. Do this before
proposing a plan or editing code. Read-only discovery may precede it only when
needed to establish facts.

The analysis must answer:

1. What is the observed problem, with concrete evidence?
2. Is it one instance of a broader class? Search for the whole class before
   fixing the first reproduction.
3. Which abstraction owns the underlying semantic fact?
4. What is the highest correct layer at which the problem can be eliminated?
5. Which current abstraction, fallback, duplicate path, or workaround is
   wrong and should be deleted rather than wrapped?
6. What is the simplest exact result, independent of compatibility with
   current generated code?
7. What are the staticness, generated-size, typecheck-memory, runtime, and
   later-translation consequences?
8. What direct source-to-output examples demonstrate the decision?
9. What independent proof and mutation test will establish the fix?
10. What broad searches and deletion checks will prove no sibling or legacy
    path remains?

Do not begin with a local patch and justify it afterward. If the analysis
reveals that the current design is wrong, replace it at its owner and delete
the superseded path.

## Governing Objective

Gotots performs complex compile-time analysis so it can emit simple, direct,
statically typed TypeScript. Generated TypeScript should resemble a competent
manual port of the Go source. Complexity belongs in semantic analysis and one
immutable plan, not in every generated call site.

Gotots targets total coverage of the selected TS-Go product corpus. It is not
a general all-Go translator, but every supported semantic class must be
principled, exact, and scalable to new TS-Go idioms. Unknown behavior fails
closed as a typed unsupported/manual-required result.

LSP and fourslash are outside the selected product scope.

## WCBUBWHB Invariants

- One semantic fact has one authoritative producer.
- Fix repeated defects at their shared semantic class, not by file, package,
  function, identifier, or first reproduction.
- There is one current implementation path. Do not add compatibility readers,
  old/new flags, fallback binding, legacy emitters, adapters around obsolete
  protocols, or dual schemas.
- Delete wrong abstractions when replacing them. Do not preserve dead fields,
  tests, comments, helpers, or facades.
- Prefer ordinary TypeScript whenever it is exact.
- Introduce custom carriers or runtime machinery only with typed, local,
  machine-verifiable necessity evidence.
- Representations own their semantics once. Callers do not retransmit zero,
  equality, clone, key, RTTI, pointer, or representation operations.
- Definitions are emitted once; references may repeat.
- No reflection, runtime spelling lookup, source-name inference, host-shape
  probing, erased-result recovery, or dynamic semantic dispatch.
- No semantic payload is recovered from `any` or `unknown`.
- Preserve Go evaluation order, storage identity, value copying, nil behavior,
  panic behavior, and initialization semantics exactly.
- Choose the best exact and performant architecture, not the one requiring the
  fewest edits to existing code.

## Generated Product Quality Is Correctness

Semantic test-passing alone is insufficient. Every generated-output change
must satisfy all four dimensions:

1. Go semantic equivalence.
2. Strict TypeScript correctness.
3. Source-shaped, statically recoverable output architecture.
4. Bounded generated size, typecheck cost, memory, and runtime cost.

A change is incomplete if any dimension is unmeasured or knowingly blocked.
Gate 16 is the complete performance gate, but source-shape, size, duplication,
and staticness checks must run from the first fixture and on every relevant
checkpoint.

## Mandatory Generated-Artifact Review

Do not review only implementation diffs or summary reports. Inspect actual
formatted generated TypeScript.

For every architecture milestone, review:

- every calibration fixture;
- the twenty largest generated bodies;
- the twenty highest source-to-output expansion ratios;
- the twenty widest generated call and type expressions;
- every new representation or exception class;
- a deterministic identity-hash sample of ordinary bodies;
- every generated file outside the calibrated size distribution;
- every byte-attribution category that changed materially.

Every technical example presents:

1. Go source;
2. semantic/compiler decision;
3. generated TypeScript;
4. why the simpler form is or is not exact;
5. byte/token/AST and performance impact;
6. proof and mutation-test evidence.

## Shift-Left Architecture Gates

Every generated-output-changing checkpoint reports absolute values and deltas
for:

- exact selected Go and generated TypeScript bytes/tokens/AST nodes;
- body expansion median, p90, p95, p99, and maximum;
- call count, source/target argument count, and hidden semantic arguments;
- receiver-call preservation;
- helper/carrier/check density by typed reason;
- definitions, references, duplicates, and artifact collisions;
- largest files, bodies, types, and calls;
- unattributed bytes;
- dynamic/erased/staticness violations;
- Gotots generation time and peak RSS;
- strict TypeScript time and peak RSS;
- representative runtime results.

Aggregate improvement cannot hide a worsening tail. Moving bytes from core to
ABI, externals, helpers, or generated types does not remove them; report those
denominators separately.

## Stop-The-Line Conditions

Stop feature work and reopen the owning abstraction immediately when:

- an ordinary call gains a hidden semantic argument;
- a method becomes a free-function call without typed exception evidence;
- a semantic definition is emitted more than once;
- an implementation/artifact identity is overwritten or ambiguous;
- output recovers semantics from `any`, `unknown`, names, or runtime shape;
- output bytes cannot be completely attributed;
- generated/source ratio or an expansion percentile exceeds its calibrated
  budget;
- total size or p95 expansion grows materially without source growth and typed
  evidence;
- typecheck time or RSS exceeds its calibrated budget;
- a local exception spreads into ordinary code;
- the same workaround class appears a second time elsewhere;
- a required architecture/performance gate remains blocked by the active
  design.

Do not respond by raising a threshold, adding an allowlist, special-casing the
largest site, or weakening a test. Audit and fix the shared owner.

## New-Mechanism Requirement

Before introducing a carrier, helper protocol, specialization, conversion
artifact, side table, or custom runtime operation, record:

- the exact Go behavior ordinary TypeScript cannot preserve;
- the smallest counterexample;
- the authoritative truth owner;
- all simpler rejected alternatives;
- affected site/region/binding-family/corpus counts;
- generated shape and unique implementation identity;
- definition and reference bytes;
- generation/typecheck/runtime costs;
- staticness implications;
- differential and mutation proof;
- reopening and deletion criteria.

If necessity is local but the mechanism is broad, the abstraction is wrong.

## Evidence And Reporting

Use exact evidence-stage and denominator names. Distinguish:

- selected;
- typed;
- IR-admitted;
- lowered;
- body-materialized;
- module-retained;
- strict-typechecked;
- semantic-class-validated;
- package-executed;
- compiler-differential-validated;
- product-assembled;
- certified.

Do not call IR admission, partial emission, subset typechecking, or local
oracles “working,” “safe,” “translated,” or “complete.” Every percentage names
its numerator and denominator. Every count disagreement is resolved by
canonical identity join, never by choosing a preferred number.

Evidence must bind to the exact clean pushed implementation/source revision.
Reviewers reproduce key totals and inspect artifacts independently rather than
trusting implementer prose or shared hashes.

## Testing And Validation

- Start with focused semantic, differential, and mutation tests.
- Run strict typechecking before executing generated TypeScript.
- Compare parser/compiler behavior against pinned Go/TS-Go semantics.
- Never weaken tests to match an implementation defect.
- Only the coordinator runs heavy full gates, whole-product typechecks, and
  large oracle suites.
- Do not stack heavy jobs. Use disk-backed `.temp/` runs, memory preflight,
  bounded concurrency, and current-test/shard breadcrumbs.
- Preserve failed-run artifacts so crashes, OOMs, timeouts, and semantic
  failures remain distinguishable and resumable.

## Work And Git Hygiene

- Work on one active feature branch; commit and push meaningful checkpoints.
- Never force-push, delete remote branches/tags, or use `git stash`.
- Use `.analysis/` for uncommitted analysis and `.temp/` for scratch work.
- Do not commit `.analysis/`, `.temp/`, `.tests/`, generated products, or local
  build artifacts.
- Use `apply_patch` for edits.
- Keep non-generated implementation files semantically focused and below 600
  lines; split by responsibility before exceeding that size.
- Before mass edits, dry-run and inspect representative examples/counts.
- Each replacement checkpoint installs one authoritative owner, switches all
  producers/consumers, and deletes the old path in the same change.

## Recurrence And Completion

At the first defect, inspect its semantic class. At the second related defect,
stop local patching and perform a broad class audit immediately.

Before calling any phase complete, confirm:

- actual generated artifacts were reviewed;
- architecture trend evidence is current;
- every new broad mechanism has necessity/cost evidence;
- duplicate definitions and artifact collisions are zero;
- hidden ordinary-call arguments are zero;
- unattributed bytes are zero;
- dynamic/erased semantic recovery is zero;
- no sibling, fallback, compatibility, or legacy path remains;
- all applicable gates pass at the exact clean pushed head.

Gotots product completion requires `18 pass / 0 blocked / 0 fail`, a current
attestation, a complete published product, and remaining required work equal to
zero.

