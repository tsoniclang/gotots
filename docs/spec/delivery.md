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

## No-Compromise Design Gate

No phase, shared abstraction, or cross-phase artifact enters implementation
until a design-only WCBUBWHB review records **no known architectural
compromise** against the mission. “No known compromise” is falsifiable design
closure, not a claim that defects are impossible. The review fails unless the
governing specification defines:

1. every closed input class and authoritative producer;
2. each identity domain and its stability/lifecycle boundary;
3. every output record, field, closed variant, and relation cardinality;
4. which concerns are orthogonal and therefore forbidden from sharing one
   field, enum, span, digest, or selection switch;
5. context, containment, ownership, and ordering for every source form;
6. exact conservation equations and independently derived joins;
7. positive, boundary, cross-product, and adversarial examples;
8. mutations that fail the real production owner rather than a synthetic foil;
9. asymptotic work/storage plus source-size, memory, typecheck, and runtime
   bounds; and
10. the superseded schema, producer/consumer paths, tests, and prose deleted by
    the replacement.

An architectural noun is not specified by a name. A discriminator enum is not
the payload it classifies; a span is not topology; a hash is not authority; a
passing count is not an identity join; one increment per visitor is not total
work; and an API described as immutable or independent must define what is
unreachable and what evidence is separately derived. If a trivial placeholder
can satisfy the literal words while omitting the intended information, the
specification fails this gate.

The review exercises the complete cross-product of representation-independent
dimensions before implementation—for example explicit/implicit definitions,
all evidence depths, local/certified structural sources, source/checked cgo
views, parent/child depth combinations, and source-spanned/bodyless forms.
Accepted authority is replaced atomically rather than amended with a second
meaning of an existing term. Implementation begins only after the specification
itself passes this gate.

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

Every numbered dependency step closes independently before work begins on its
consumer. Closure is revision-bound and requires all of the following as
applicable:

1. focused owner/consumer contract tests;
2. independently derived exact identity joins at each changed boundary;
3. production-path mutations for every claimed failure class;
4. inspection of representative real artifacts and the twenty-item affected
   size/complexity tail;
5. staticness, source/generated size, peak memory, wall time, typecheck, and
   runtime measures with absolute values and parent deltas;
6. the guarded broader suite covering every migrated consumer; and
7. broad searches proving deletion of the superseded schema, producer,
   consumer, fallback, test, and prose.

Evidence from an earlier revision, a narrower substitute, or an intended
phase-exit run cannot close the step. A failed check reopens that step's
authoritative owner and blocks all dependent steps. Phase exit reruns the union
of the step gates; it does not provide the first comprehensive verification of
an earlier step.

## Dependency-Ordered Implementation

### Phase: Language Inventory

Build the selected-Go-version construct catalog before translation code.

Required exit evidence:

- grammar, AST node types, tokens, built-ins, directives, and version features
  reconcile exactly;
- context-resolution roles and variants are enumerated;
- implicit operations are cataloged;
- `gotots inspect constructs` reports two unrelated multi-module projects,
  exact resolved closures, and disjoint full-semantic, declaration-contract,
  external-boundary, and intrinsic evidence sets;
- definition kinds and identity constructors are pinned; the complete
  definition census includes function/method declarations, function literals,
  package initializers, bodyless obligations, and implicit executable work
  before evidence-depth selection;
- each selected source file/synthetic owner has exactly one owner region; each
  source-file region has one normalized sparse containment graph; and each
  definition has exactly one site, header region, execution boundary, and
  separate definition selection;
  every retained occurrence identity has exactly one canonical payload, and
  owner/header/boundary/containment/executable relations reference it rather
  than copying context fields;
  source definitions use construct-root `DefinitionID`s rather than body-entry
  identities, and later concrete `ImplementationID`s remain a separate domain;
- header and execution content addresses are disjoint: body-only edits preserve
  header digests, while header-only edits preserve execution-entry digests;
- the catalog classifies every child edge as header, execution entry, nested
  definition site, or ordinary executable child, with no second edge table;
  parent-directed definition classification includes declaration class/token,
  so package `const` and `var` `ValueSpec` nodes cannot collapse into the same
  implementation disposition;
- header occurrences exist once at every depth; retained source occurrences or
  typed implicit operations cover exactly full-semantic definitions, while
  declaration-contract and external definitions retain exact boundaries and
  zero executable occurrences;
- files and packages may aggregate several definition depths without a default;
- definition-kind/evidence-depth compatibility exact-joins the closed matrix;
  bodyless obligations cannot masquerade as full semantic bodies and implicit
  full definitions require an exact typed executable graph;
- the compilation request selects one digest-bound provider contract, and every
  definition records the exact rule/evidence selecting its provider and depth;
- conditional rules consume only closed identity-keyed selection facts from one
  producer over declared finite candidate sets; those facts are reused by Stage
  2 and never recomputed by scope or the typed frontend;
- structural-source planning is the validated union of exact-definition,
  exact-package, namespace, and conditional rule requirements; every possible
  full-semantic candidate has recursive evidence before executable inventory,
  and circular/unbounded conditional acquisition fails contract validation;
- package owner, provenance, acquisition, and language disposition exact-join
  independent selected-toolchain metadata, including standard-library,
  dependency, replacement, `unsafe`, and pseudo-package cases;
- every Go, non-Go, embed, overlay, and checked-view input has a stable typed
  identity separate from its acquisition path, and relocated inputs preserve
  those identities;
- compiler capability, selected toolchain, module directive, and effective
  per-file language versions remain distinct, with version-gated occurrences
  verified against their own file versions;
- an injected unknown form fails;
- no target or acceptance-corpus package is imported;
- complete-toolchain catalog coverage runs as a separate streaming audit and
  produces a versioned artifact consumed without rescanning standard-library
  bodies during ordinary compilation;
- the audit exact-joins selected-byte, build-configuration, provider-contract,
  toolchain, catalog, per-file count, and aggregate-count evidence;
- provider artifacts carry the complete logical
  owner/containment/definition/site/header/boundary graph in independently
  content-addressed package shards; the resident manifest carries only exact
  package/file membership, definition identities, header/boundary counts,
  requested selection facts, and shard admission data; each file uses exactly
  one local or certified production graph, and ordinary compilation consumes
  it without rescanning provider interiors or retaining all provider detail;
- source owns bytes/acquisition/transient checker lifetime,
  `internal/scope/contract` solely owns the closed provider/rule/fact-request
  schema,
  `internal/scope/sourceplan` solely owns the pre-graph local/certified plan,
  `internal/language/structure` solely owns the depth-independent definition
  graph, `internal/language/selectionfacts` solely owns closed preselection
  facts, `internal/scope` solely owns per-definition selection,
  `internal/language/executable` solely owns full executable occurrence/role
  expansion, and `internal/language/frontend` solely owns typed semantic
  resolution; architecture gates reject ownership leakage or private
  edge/variant tables;
- finalized artifacts expose one
  owner/containment/definition/site/header/boundary API at every depth plus
  exactly one executable region per full definition, retain no non-full body
  AST or body-indexed `types.Info`, and hold no uniform-full/mixed consumer
  split;
- exact joins prove source/synthetic owners↔owner regions,
  source-file regions↔normalized containment graphs and sites↔complete paths,
  census↔definitions, definitions↔sites and rooted acyclic containment,
  definitions↔headers, catalog header occurrences↔header regions,
  definitions↔boundaries, definitions↔selections, full
  definitions↔source-or-implicit executable regions, non-full definitions↔zero
  executable occurrences, and local/certified graph parity;
- the outer-full/child-non-full and outer-non-full/child-full matrices both
  preserve each definition and site exactly once while retaining executable
  occurrences only for the selected full definition;
- production work/storage satisfy the architecture's linear bound plus only
  named `O(n log n)` sorts; a counted linear-scan mutation in a real production
  lookup and a copied-per-site-path mutation both fail the gate;
- ordinary provider consumption constructs its whole-universe definition
  census from manifest identities without opening package shards, exact-joins
  each projected shard to that census and its requested facts, and keeps at
  most one provider package resident; eager all-shard decode, detail duplicated
  into the manifest, repeated all-fact scans, and a multi-package cache each
  fail their owning gate;
- ordinary consumption authority comes from an independently selected certified
  digest, and audit certification binds the overlay/build-input projection that
  can change audited membership or selected bytes; and
- package, definition, site, header occurrence, boundary, executable
  occurrence, provider-artifact byte, construction-work, elapsed-time, and peak
  RSS measures have reviewed absolute and parent-delta bounds, including the
  twenty largest header/provider artifacts; provider production and ordinary
  consumption are measured separately, and ordinary RSS enforces
  `local detail + identity census + requested facts + largest provider package`
  rather than total provider-detail residency.

### Phase: Typed Frontend And Semantic Model

Consume Stage-1's definition graph, full executable occurrences, and
parent-assigned grammatical roles; materialize target-independent definition
semantics and executable operations from the one transient checker graph or one
certified provider semantic authority, reusing preselection facts, before
finalization. This phase does not rebuild loading, owner/containment structure,
definition identities, sites, headers, boundaries, selections, preselection
facts, or Stage-1 visitors.

Required exit evidence:

- every definition has exactly one `DefinitionSemantics` record from the
  transient checker graph or one certified provider semantic authority,
  with a closed authority witness/input digest and covering its
  receiver/signature, declared names/types, initializer/bodyless obligation, or
  implicit-operation meaning;
- when checker and provider semantic evidence both exist they exact-join as
  corroboration and exactly one selected authority enters the model;
- every retained owner/header/boundary/executable occurrence has exactly one
  closed resolution domain and exactly one legal
  `OccurrenceResolution` (`StructuralOnly`, definition component, declaration,
  binding, type, operation, or explicit unsupported), and no occurrence is
  silently consumed or omitted; operations occur only in executable domains,
  while non-executable compile-time syntax is positively accounted for by its
  exact structural coverage target naming its owning declaration/type fact
  rather than a generic fallback;
- definition semantics preserve the exact declaration cardinality and order:
  one declaration for ordinary function/method and bodyless definitions, zero
  for `func _()`, `func init()`, literals, and implicit definitions, zero-or-more for a
  multi-name package initializer after blank names are removed (including zero
  for `var _ = f()`), and one for a typed synthetic definition; synthetic adapters
  require a signature while synthetic type/data definitions forbid one;
  evidence depth and provider class are never encoded as semantic forms;
- every full-semantic executable region is completely resolved; every
  semantically executable occurrence resolves to an operation or explicit
  unsupported record, catalog-authorized structural-only occurrences remain
  explicit, every typed implicit executable entry has one closed non-source
  operation identity without a fabricated span, and every non-full definition
  has zero executable operations;
- module, standard-library, and toolchain package identities are constructor-
  validated without machine paths or fabricated module ownership;
- predeclared declaration payloads are owned once by the `builtin` language
  pseudo-package, exact-joined to the pinned catalog and `types.Universe`, and
  are referenced rather than copied by ordinary packages;
- predeclared and `unsafe` builtin declarations use their closed catalog
  identities rather than a fabricated ordinary signature; every builtin call
  retains its exact call-site semantic evidence, and the selected toolchain's
  `unsafe` builtin set exact-joins the append-only catalog;
- each type parameter is canonically owned by one generic declaration or
  declaration-less definition plus binder role and ordinal; local lexical
  bindings reference that type identity, while imported export-data and
  instantiated checker forms materialize without foreign syntax, a fabricated
  local binding, or checker-object identity;
- ordinary package semantics are independent of provenance; semantic records
  carry the resolved package identity/provenance but contain no output path or
  implementation-owner decision, which belongs to later planning;
- shadowing, scopes, generic binders, method identity, and source spans are
  exact; binding ordinals are assigned once over the complete direct
  definition and implicit-definition binding set for each canonical
  scope/role before projection, unnamed signature slots exact-join their
  implicit object to the checker signature tuple and structural field, and
  omitted siblings remain identity evidence so independent counters cannot
  collide;
- generic-owner derivation is package-scoped and dependency ordered: current
  declarations are indexed once, an exact imported checker object registers
  its origin type/callable/receiver parameters before its type is consumed,
  and no per-package whole-universe scope walk or position/name fallback
  exists;
- any nested definition whose enclosing definition is non-full can consume
  exact enclosing local type/alias/constant declarations for its required
  definition semantics, and a full nested definition can additionally consume
  exact enclosing lexical bindings, through direct `Info.Defs` plus Stage-1
  transient occurrence/scope/definition anchors; only the required semantic
  closure is emitted, stable ordinals include omitted siblings, and the
  excluded parent contributes zero executable operations and no finalized AST;
- local semantic production writes definitions, resolutions, bindings, and
  operations directly into one normalized draft, derives one pre-seal fixed
  point over every admitted declaration/type reference, transfers each selected
  declaration and type into that same draft once, and seals once; provider
  production starts from every package-owned declaration, mixed-depth checker
  production starts from admitted local records, and no post-seal projection,
  closure repair, or second complete package representation exists;
- repeated spellings of an equal unnamed structural type exact-join their
  declaration occurrences to one canonical type-owned member target without
  selecting one spelling as payload authority; package/local/predeclared/
  synthetic objects have one standalone declaration record, fields and methods
  have zero, and every member reference resolves through the canonical owner
  type after pointer removal, alias collapse, generic-origin collapse, and
  promoted-owner selection;
- interface type sets distinguish universe, finite normalized terms, and empty,
  and branch records distinguish an exact same-definition control-target
  operation from an optional label binding;
- context fixtures cover call/conversion, map/receive/assert comma-ok,
  selectors, assignments, composite literals, inferred-length `[...]T`
  arrays, explicit constant array lengths, local/package `const`
  initializers, and range; inferred array ellipses and explicit array-length
  expression subtrees resolve through checker-derived compile-time coverage
  and never fabricate runtime operations, while the same binary expression in
  an ordinary assignment remains runtime;
- nested implicit effects preserve per-operand multiplicity (`f(a, b)` can
  carry two distinct value-copy effects) rather than collapsing to one entry
  per implicit kind;
- semantic records are immutable and constructor-validated;
- immutable semantic packages expose counts, ordered visitors, and canonical
  identity lookups only; declaration-target lookup is a closed validated view
  over one standalone record or one type-owned field/method component and
  creates no resident member index, record slice, or complete cloned record
  class; and
- each logical semantic package and its wire shard use one normalized
  representation: typed package-local tables own canonical identity components
  once and composite identities reference their constituent tables, compact
  records store only their active closed-variant payloads, and variable
  relations use validated typed ranges; public semantic values are transient
  one-record projections, package-local references never escape, the prior
  denormalized schema/reader/writer are deleted; semantic-detail shards use the
  explicit reflection-free binary schema while only the bounded manifest may
  remain JSON; single-authority admission transfers normalized stores without
  record projection/re-interning and mixed admission uses bounded ordered
  cursors; dictionary-domain, range, active-payload, version, truncation,
  trailing-byte, round-trip, mutation, struct-size, encoded-size,
  largest-package, and whole-compilation cost gates pass; and
- the sealed package retains the compact identity-component graph rather than
  expanded composite identity slices, admission and reachability consume the
  compact records directly, and member-target census generation streams
  without a rendered or package-wide derived member index; and
- no TypeScript representation appears in the model;
- Stage-1 structural provider evidence and Stage-2 semantic provider evidence
  remain separate digest-bound authorities; the semantic artifact binds the
  exact structural digest and is independently reproduced before acceptance;
- checker-derived local and certified-provider semantic detail are both
  package-sharded behind one projection contract; local production writes and
  releases one validated package at a time while preserving
  `CheckerAuthority`, ordinary Stage-2 verification projects every package
  carrying checker authority exactly once and opens no provider-only shard,
  and at most one final logical package is resident; a mixed projection streams
  both authorities into one ownership-transferred package draft, exact-joins
  overlaps after authority removal, retains no package-wide wire tree or
  separate local/provider/merged package copies, and passes the frozen
  largest-package and whole-compilation RSS/record/byte budgets;
- frontend package preparation obeys that same package-at-a-time boundary:
  whole-universe state contains no resident collection or index of derived
  package inputs, each package's occurrence/child/containment/object/draft state
  becomes unreachable after its shard is written, and executable-only
  occurrences are obtained through package/file-scoped indexes rather than a
  complete-set scan per package;
- canonical type/declaration/binding/operation identities remain identical
  across relocated workspaces, relocated module caches, local checker
  production, and certified-provider consumption; and
- every computational identity sort, search, lookup, deduplication, and join
  uses the identity package's allocation-free structural order; identity
  serialization occurs only at artifact and diagnostic boundaries, and no
  rendered-key cache or consumer-owned component order exists; and
- the provider semantic schema is bumped atomically for the type-owned-member
  contract, the prior reader/writer is deleted, per-package standalone
  declaration and derived member-target counts/digests exact-join, and no
  serialized member `Declaration` is accepted; and
- measured construction uses the closed named-pass ledger, is linear with a
  fixed coefficient plus named canonical sorts, and reports
  definitions, resolutions, declarations, bindings, types, operations,
  unsupported records, artifact bytes, largest shards/records, wall time, and
  peak RSS with parent deltas and top-twenty tails.

### Phase: Facts And Planning

Build effect/storage/copy/generic/interface/object/init/call/reachability
analyses and one immutable `ProgramPlan`.

Required exit evidence:

- each fact has one owner and seals before use;
- semantic reachability starts from explicit executable/API/test/reflection/
  extension roots, records exact root/exclusion witnesses, and exact-joins every
  typed call/initialization/function-value/generic/dynamic/registration/external
  edge before planning;
- planning and lowering consume only the semantically reachable set; a mutation
  that plans the whole selected package closure fails size, identity, and
  reachability gates;
- plan records are total and atomic;
- no emitter decision or post-plan mutation exists;
- ordinary examples select direct plans;
- mutation of a fact edge changes or invalidates the plan predictably; and
- plan cost estimates reconcile with expected definitions and references.

### Phase: Typed TypeScript And Source Shape

Pin the exact TS-Go schema-level AST contract, generate its typed Go bindings,
factories, predicates, visitors, transforms, and validator, then implement the
sole formatter, source-file modules, and direct plan-to-AST lowering for
ordinary declarations, expressions, statements, functions, receivers, and
control. There is no handwritten target AST and no text-emission intermediate.

Required exit evidence:

- `schema/tsgo/` binds one TS-Go revision, every upstream schema input and
  digest, and the generator version; a clean regeneration exact-joins node
  kinds, abstract categories, fields, optionality, factory parameters, child
  order, and visitor/transform edges in both directions and produces no diff;
- every generated AST artifact is admitted through the generated validator;
  malformed kind, missing/extra field, wrong optionality, wrong child order,
  hand-added node shape, generic property-bag, and raw-source-node mutations
  fail before formatting;
- calibration fixtures match reviewed hand-port AST shape;
- ordinary calls have no hidden semantic arguments;
- receiver methods remain methods;
- native, checked-thunk, and explicit-receiver method plans pass nil,
  evaluation-order, value-copy, exact-selection, and interface-dispatch
  differential/mutation fixtures;
- generated method values and expressions use typed lambdas/functions with no
  `call`, `apply`, `bind`, or prototype invocation;
- lowering returns exact TS-Go AST nodes and never source strings, templates,
  token streams, raw fragments, ESTree nodes, or TypeScript compiler
  JavaScript objects; no TypeScript text is generated or patched outside the
  sole formatter;
- formatted calibration output is reparsed by the pinned TS-Go parser, its
  schema-level structural projection exact-joins the constructed AST projection
  modulo one closed field-by-field classification of parser-derived ranges,
  trivia, synthetic-only metadata, and formatter-normalized parentheses, and
  formatter cases are differentially checked against the pinned TS-Go printer;
- full output parses, strict-typechecks, and passes the Tsonic target-subset
  checker; and
- AST-node counts, source bytes, formatter time/RSS, strict-typecheck time/RSS,
  and top-twenty expansion tails are attributed and gated before expanding
  coverage.

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

Before any review, correction, or implementation directive is sent, its author
performs the separate Directive Review Gate defined in repository governance.
The reviewed message binds an exact revision and one active phase, classifies
each item as current or deferred, compares itself with the prior directive,
states any supersession explicitly, checks literal implementer consequences and
cost, and exposes one frozen endpoint. Later-phase roadmap items cannot become
active merely to make a checkpoint larger. A directive that fails this review
is revised before transmission.

An accepted directive remains frozen until its endpoint. New reproducible
evidence may amend the smallest affected portion only after another Directive
Review Gate; unaffected criteria and later-phase boundaries remain unchanged.
This review protocol is the control against fix-review-expand loops.

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
- GoToTS translated from its own Go source compiles through Tsonic without a
  JavaScript-only dynamic-invocation escape path.
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
