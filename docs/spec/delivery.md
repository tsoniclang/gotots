# Cleanroom Delivery And Completion

## Why A Cleanroom Replacement

Existing implementation evidence demonstrates useful semantics and exposes valuable
counterexamples, but its generated architecture allowed repeated aliases,
hidden generic operation plumbing, consumer-owned interfaces, oversized
modules, and per-call dispatch expansion. A `7.8x` source-size result was not a
late optimization problem; it showed that semantic ownership and planning were
at the wrong layers.

The replacement therefore begins from this specification and the selected Go
toolchain, not from the old compiler's package boundaries. Minimal edits to the
old pipeline are not a goal. Emitted APIs may change. Existing implementation
is retained only when independent review shows that the same component would be
chosen in a cleanroom design.

## Cleanroom Admission Rule

No existing file or WIP commit is imported wholesale. A candidate component is
admitted only if it:

- has a project-independent semantic owner in `architecture.md`;
- contains no acceptance-corpus condition, fallback, duplicate state, erased
  recovery, or generated-text coupling;
- has a closed typed input/output contract and validating constructors;
- does not import a forbidden higher layer;
- has focused positive, negative, differential/contract, and mutation tests;
- produces source-shaped output or no output;
- passes size, memory, staticness, and determinism review; and
- replaces rather than wraps any displaced component.

Every admitted unit records origin revision, retained behavior, rejected old
behavior, new owner, test evidence, and deletion search. Copying code without
this record is prohibited.

Likely evidence worth mining includes:

- minimized Go counterexamples and differential oracle cases;
- canonical identity test cases and known collision examples;
- workspace/toolchain fingerprinting ideas;
- TypeScript parser/TypeChecker service protocols;
- customer extension compatibility fixtures;
- manual-regeneration and graph scenarios;
- generated-output size/shape measurements; and
- semantic discoveries from the pushed object-model WIP.

Old IR, emitter, translator, object-model heuristics, repeated ABI machinery,
manual registries, prefix classifiers, gate waivers, and corpus-specific plans
are not foundations. Their behavior may become a test; their architecture is
deleted.

## Atomic Replacement Policy

Development occurs on one feature branch, but production has one path:

1. Preserve a reproducible old-baseline artifact and its failures as evidence.
2. Extract and independently review reusable tests/data.
3. Delete the old production compiler path.
4. Install the cleanroom package boundaries and a minimal fail-closed CLI.
5. Grow support by catalog kind and semantic operation.

There is no `v2`, compatibility binary, old/new flag, alternate reader, or retry
through the old emitter. Git history is the archive. Temporary comparison tools
live outside production packages and are deleted before acceptance.

An architectural replacement lands with its one owner, all producers
and consumers, and deletion of the old fields/helpers/tests/comments in the
same coherent change. Split ownership is never a checkpoint.

## Dependency-Ordered Implementation

### Phase: Language Inventory

Build the selected-Go-version construct catalog before translation code.

Required exit evidence:

- grammar, AST node types, tokens, built-ins, directives, and version features
  reconcile exactly;
- context-resolution roles and variants are enumerated;
- implicit operations are cataloged;
- `gotots inspect constructs` reports a small arbitrary module;
- an injected unknown form fails; and
- no target or acceptance-corpus package is imported.

### Phase: Typed Frontend And Semantic Model

Implement canonical identities, workspace loading, parent-assigned contextual
visitors, and the target-independent semantic records.

Required exit evidence:

- every selected occurrence has one typed semantic operation or unsupported
  record;
- shadowing, scopes, generic binders, method identity, and source spans are
  exact;
- context fixtures cover call/conversion, map/receive/assert comma-ok,
  selectors, assignments, composite literals, and range;
- semantic records are immutable and constructor-validated; and
- no TypeScript representation appears in the model.

### Phase: Facts And Planning

Build effect/storage/copy/generic/interface/object/init/call/reachability
analyses and one immutable `ProgramPlan`.

Required exit evidence:

- each fact has one owner and seals before use;
- plan records are total and atomic;
- no emitter decision or post-plan mutation exists;
- ordinary examples select direct plans;
- mutation of a fact edge changes or invalidates the plan predictably; and
- plan cost estimates reconcile with expected definitions and references.

### Phase: Typed TypeScript And Source Shape

Implement the target AST, formatter, source-file modules, and direct lowering
for ordinary declarations, expressions, statements, functions, receivers, and
control.

Required exit evidence:

- calibration fixtures match reviewed hand-port AST shape;
- ordinary calls have no hidden semantic arguments;
- receiver methods remain methods;
- no TypeScript text is generated outside the formatter;
- full output parses and strict-typechecks; and
- size/shape gates run before expanding coverage.

### Phase: Representations And Dynamic Semantics

Implement only proven representation needs: values/copies, pointers, slices,
maps, byte-sensitive strings, interfaces, generics, class/object families,
defer/panic, channels, and initialization.

Required exit evidence for each class:

- smallest Go counterexample and simpler rejected forms;
- whole-program fact and atomic plan;
- one owned definition with bounded references;
- no erased payload or source-name dispatch;
- no per-call expansion proportional to implementer or instantiation count;
- differential/property and mutation tests; and
- measured bytes, typecheck cost, and runtime.

Classes are enabled corpus-wide only after their calibration fixtures pass.

### Phase: Runtime And Go Standard Library

Build the minimal runtime and generate the selected Go standard-library
declaration workspace. Manually complete reachable standard-library behavior
using the same body-hash workflow.

Required exit evidence:

- runtime necessity ledger has no broad unproven mechanism;
- standard-library API is derived from the selected `GOROOT` with zero
  hand-maintained signature drift;
- definitions are canonical and not duplicated per project;
- every reachable placeholder has a manual implementation or blocks;
- package and cross-package Go-versus-TS contract tests pass; and
- a Go-version update produces exact structural/manual deltas.

### Phase: Completion, Externals, Extensions, And Graph

Implement mixed-file ownership, post-format hashes, empty-root regeneration,
external stubs, the retained customer extension contract, full graph traversal,
status/reset/prune, and atomic assembly.

Required exit evidence:

- no user-authored manual registry exists;
- every prior generated unit is discarded on regeneration;
- compatible manual bodies survive while surrounding generated code changes;
- signature conflicts fail structurally;
- all graph node/edge classes have mutation fixtures;
- every retained unit has a root witness;
- unreachable manual work is reported and safely pruneable; and
- the existing extension compatibility suite passes unchanged.

### Phase: Complete Go Coverage

Run arbitrary projects and expand support by semantic class, never by corpus
site. Work order is highest product reachability and occurrence count, except a
stop-the-line architecture defect takes priority.

For each missing class:

1. inventory and broad-search all occurrences;
2. minimize source cases and context variants;
3. decide semantic operation and owner;
4. add fact/plan only if necessary;
5. emit the simplest exact TypeScript;
6. differential/property/mutation test;
7. inspect ordinary and worst generated artifacts; and
8. recount by canonical identity.

The phase closes only when the selected Go version has zero unknown catalog
kinds and every selected occurrence is automatic, accepted manual, or resolved
environment behavior.

### Phase: Independent Product Verification

Complete all verifiers and run the eighteen gates against:

- construct micro-projects;
- at least two unrelated realistic Go projects selected before inspecting
  their generated output;
- the manually implemented standard library and external providers;
- extension-free and extension-enabled products; and
- the first large acceptance corpus, `typescript-go`.

The second unrelated project must pass before a representation discovered in
the first corpus is treated as generally validated.

### Phase: Upgrade And Publication

Perform a real upstream Go project revision and Go toolchain-version upgrade.
Regenerate from an empty root, reconcile manual/extension changes, rerun all
gates, and atomically publish.

Required endpoint:

```text
18 pass / 0 fail / 0 blocked
reachable placeholders = 0
unclassified constructs = 0
identity/artifact collisions = 0
duplicate semantic definitions = 0
unattributed generated bytes = 0
source-project-specific compiler branches = 0
```

## Go Version And Project Upgrade

Every upgrade follows one deterministic procedure:

1. fingerprint old and new toolchains/workspaces;
2. reconcile the language catalog and fail on new unknown forms;
3. rebuild scope, identities, semantic model, facts, and plans from source;
4. identity-diff declarations, occurrences, implementations, tests, contracts,
   and extensions;
5. generate a fresh empty-root baseline;
6. structurally overlay compatible manual bodies;
7. rebuild the complete graph and classify obsolete work;
8. run all gates and compare cost distributions; and
9. publish only the certified immutable product.

An added use of an implemented semantic class needs no local patch. A new class
receives one generic construct/semantic decision. Removed source does not make
manual history reachable.

## Team Execution Discipline

The team executes the dependency order continuously without requesting
approval for routine next steps. It pauses only for a genuine product decision
not settled by this specification, a destructive remote operation, or an
external credential/environment blocker.

Checkpoint reports are concise and exact. They include:

- branch and clean pushed head;
- completed phase exit criteria;
- canonical denominators and parent deltas;
- actual Go and generated TypeScript examples;
- focused/full tests and exact gate states;
- source-shape, size, TypeScript RSS/time, generation, and runtime changes;
- unresolved identities and their owners; and
- broad deletion/leakage searches.

Repository `AGENTS.md` and `CLAUDE.md` contain the same WCBUBWHB and execution
rules and must remain byte-identical, verified by a repository gate.

“Complete,” “working,” and percentages are forbidden without the exact stage
and numerator/denominator. A locally green fixture, IR-admitted body, emitted
subset, or strict-typechecked subset is not product completion.

Only one coordinator runs heavy full-corpus jobs. Runs are disk-backed,
bounded, identifiable, and resumable. Failed artifacts are retained so timeout,
OOM, crash, semantic mismatch, and test failure remain distinguishable.

## Final Deletion Audit

Before publication, broad searches and import-graph checks prove deletion of:

- all displaced compiler entry points and alternate-path flags;
- source-project-specific package/type/method checks;
- old IR/emitter/object-model/ABI paths not admitted under the cleanroom rule;
- fallback identities, spelling dispatch, prefix classifiers, and retries;
- parallel maps or side tables for atomic semantic records;
- per-consumer aliases, repeated vtables, and per-call implementer switches;
- hidden generic dictionaries on ordinary calls;
- manual JSON attachment and file-level ownership;
- preservation/copying of stale generated source;
- text-based reachability and extension anchors;
- casts, erased recovery, suppressions, waivers, and allowlists;
- stale tests, comments, ADRs, reports, and documentation claiming an old path
  is current; and
- scratch comparison code and migration facades.

The audit runs as a machine gate plus independent review of final code and
artifacts.

## Final Acceptance Checklist

- GoToTS accepts arbitrary Go workspaces without corpus-specific semantics.
- The construct catalog is exhaustive for the declared Go version.
- Context-sensitive meanings are resolved before lowering.
- Every semantic fact and implementation has one owner and identity.
- Planning is whole-program, immutable, total, and independently checked.
- Ordinary generated TypeScript is direct, class/method shaped where exact,
  strict, ESM, and near the reviewed hand-port size.
- All nonordinary output has local typed necessity and cost evidence.
- Runtime definitions are minimal and singular.
- Standard-library APIs are toolchain-derived and reachable behavior is
  manually complete.
- Source-available dependencies translate; true externals have exact contracts.
- Generated/manual units coexist safely without user metadata.
- Regeneration destroys old generated output and preserves compatible manual
  work structurally.
- Full graph traversal explains every retained and unreachable manual unit.
- Pruning is explicit, dry-run-first, and hash-safe.
- The existing customer extension architecture remains compatible and fully
  participates in evidence, graph, tests, and publication.
- Actual artifacts pass independent parity, strictness, differential,
  architecture, size, memory, runtime, upgrade, and determinism review.
- All eighteen gates pass at the exact clean published revision.
