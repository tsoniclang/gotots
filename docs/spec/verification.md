# Verification And Architecture Gates

## Proof Model

Tests are necessary but no single passing suite proves translation. GoToTS uses
independent, joined evidence:

1. exhaustive construct/context classification;
2. complete semantic-model and plan validation;
3. independent source-Go versus generated-TS declaration/shape comparison;
4. strict TypeScript parsing, resolution, and typechecking;
5. semantic-class differential and property oracles;
6. per-implementation mapping and execution evidence;
7. selected project tests and whole-project differential behavior;
8. standard-library, external, manual, and extension contract tests;
9. deterministic regeneration and source-upgrade proof; and
10. source-shape, size, memory, runtime, and publication gates.

Evidence from two components sharing the same canonicalizer or plan builder is
not independent proof. A verifier must derive its observed side from final
artifacts using a different parser/extractor and join by canonical identity.

## Specification-Adequacy Proof

Before implementation of a phase or shared cross-phase artifact, Gate 01
requires a revision-bound design attestation against the No-Compromise Design
Gate. The attestation identifies the exact governing paragraphs and records the
closed inputs, owner, schema/variants/cardinalities, lifecycle, orthogonal
concerns, conservation joins, downstream consumers, examples, mutations,
complexity/cost bounds, and deletion set. Missing evidence fails; it cannot be
reported as a future implementation detail.

The adequacy review is adversarial. It attempts to satisfy the prose with a
discriminator-only record, zero/empty payload, shared identity for two
orthogonal facts, count-only comparison, depth- or provenance-selected
topology, synthetic complexity counter, and verifier sharing the producer's
critical derivation. If any such substitute can pass literally, authority is
replaced before implementation. A phase-exit review repeats this check against
actual artifacts so code cannot reveal that the accepted schema had a second
meaning.

## What It Means To Validate A Body

Every reachable implementation has an `ImplementationID` and a ledger joining:

- source `DefinitionID`, definition site/header, execution boundary, and exact
  source spans;
- construct occurrences and semantic operations;
- whole-program facts consumed;
- immutable plan records;
- generated or manual typed AST unit;
- formatted artifact and post-format hash;
- strict-TypeScript result;
- semantic-class oracle coverage;
- executed Go and TS test/workload events; and
- current reachability witnesses.

A body is not validated merely because IR was built or TypeScript compiled.
Certification requires:

- a bijective source-operation -> semantic-operation -> planned-operation ->
  emitted-AST join with no unexplained or orphan operation;
- all lowering classes used by the body passing their differential/property
  suites, including interaction dimensions such as evaluation order, aliasing,
  copying, nil, panic, and boundaries;
- strict typechecking of the fully materialized module closure;
- execution by a direct generated harness, selected original Go test, or
  project workload whose event ledger names that implementation; and
- no unresolved reachable dependency, placeholder, extension, or external
  obligation.

Execution samples cannot mathematically prove all inputs. Property generation,
boundary cases, metamorphic relations, original tests, and whole-project
differential workloads reduce that risk. Reports must call this machine
certification, not formal proof.

Every reachable public implementation must execute at least once. Every branch,
panic edge, return form, defer path, interface target, and representation
boundary must either execute in retained evidence or have an explicit stronger
class proof joined to that exact operation. Unexecuted interactions remain
blocking.

## Eighteen Gates

All gates are `pass`, `fail`, or `blocked`. Publication requires `18 pass / 0
fail / 0 blocked` at one exact input and implementation revision.

| Gate | Required positive evidence |
|---|---|
| 01 Input integrity | clean source/compiler trees, exact tool/config fingerprints, focused/full Go tests, bounded runner health |
| 02 Language catalog | selected Go grammar/tokens/built-ins and edge classes reconcile; every definition/header/full executable occurrence/non-full boundary classified; unknown count zero |
| 03 Scope closure | complete package closure, structural-source plan, and per-definition evidence-depth partition exact-join independent toolchain/environment/provider metadata |
| 04 Identity and census | owner/containment regions, definitions, sites, headers, boundaries, selections, and executable regions plus later semantic/implementation/artifact multisets join exactly; collisions, overwrites, and unexplained deltas zero |
| 05 Independent parity | independently derived definition semantics and occurrence resolutions match typed Go evidence; later independently parsed generated declarations and operation shapes match those records |
| 06 Plan totality | all facts and semantic reachability sealed; every reachable operation has one validated plan or explicit blocking state, no unreachable definition is planned, and necessity records are complete |
| 07 Artifact ownership | implementations, generated AST units, files, definitions, markers, hashes, and provenance join one-to-one |
| 08 Staticness | no erased recovery, dynamic semantic lookup, unsafe casts, unplanned dispatch, or undeclared target operation |
| 09 Regeneration and graph | two clean generations match; semantic reachability exact-joins planned implementations, and post-completion implementation reachability, manual reconciliation, relocation, and atomic replacement pass |
| 10 Strict TypeScript | every materialized runtime, stdlib, external, generated, manual, and extension module parses/resolves/typechecks together and satisfies the Tsonic target subset |
| 11 Semantic-class oracles | every implemented operation class passes Go-vs-TS differential, property, boundary, and mutation tests |
| 12 Implementation execution | every reachable implementation and required interaction has retained execution evidence; selected Go tests reconciled |
| 13 Project differential | complete generated applications match Go on output, diagnostics, state, panic, files, and allowed nondeterminism |
| 14 Environment contracts | reachable runtime, standard-library, external, platform, callback, and concurrency obligations pass contract tests |
| 15 Completion and extensions | no reachable placeholder/conflict; manual workflow and existing customer extension suite pass assembled-product tests |
| 16 Shape and performance | calibrated source shape, byte attribution, duplication, TypeScript RSS/time, generation cost, and runtime budgets pass |
| 17 Upgrade portability | a real Go source/toolchain upgrade and relocated checkout regenerate deterministically with exact manual/extension deltas |
| 18 Publication | verified immutable product is atomically promoted; manifest, signatures, artifacts, and attestation all match |

A gate cannot return pass because its prerequisite is missing. It returns
blocked with the exact missing identity set. A classifier or allowlist cannot
turn TypeScript errors or missing evidence into blocked; errors fail.

## Package-Universe And Provenance Proof

The production loader's complete package set is independently exact-joined by
canonical package identity against `go list -deps -json` executed with the same
Go binary, working files, overlays, flags, environment, and roots. Separate
`go list std` and `go list cmd` runs provide authoritative set membership. The
verifier independently derives and compares those memberships, `Standard`,
`Goroot`, module path/version, `Main`, replacement, vendor selection, compiled
files, imports, and package errors. Counts without both one-sided identity
lists are not proof.

One compilation owns one coherent `go/types` graph. For every import edge, the
importer's package/type objects exact-join the imported package record; mixing
objects from separate loader calls fails. The resolved package closure is then
partitioned independently into requested roots, full-semantic source,
declaration contracts, external boundaries, and intrinsics. These sets are
never inferred from which files happened to be parsed or traversed.

Fixtures include, in one verified closure:

- workspace packages from two `go.work` modules;
- a normal versioned dependency, a vendored dependency, and a local
  replacement whose physical directory resembles workspace source;
- a dotless third-party module path, proving spelling does not imply standard;
- standard-library and non-standard `GOROOT` packages;
- `builtin`, `unsafe`, and cgo dispositions; and
- overlays plus relocated workspace, module cache, and `GOROOT` roots.

The fixture matrix also contains two workspace modules with different `go`
directives and files with different effective language versions. It proves that
each version-gated occurrence is admitted or rejected against its own file
version, while the compiler catalog maximum and selected toolchain version
remain separate evidence.

The join proves that relocation and acquisition changes do not alter semantic
identity, while module-version and source-contract changes produce the exact
expected deltas. Output-routing tests then prove standard declarations occur
once under `gostdlib`, source-available dependencies occur under the ordinary
mirrored tree, pseudo-packages do not become ordinary modules, and only exact
external obligations enter an external contract root.

Required mutations flip `std`-set or `Goroot` evidence, treat `Module == nil` as
an error, classify by import spelling, misclassify a `cmd` package as standard
library, relabel a local replacement as a workspace module, omit a transitive
dependency, route a standard declaration into product output, and duplicate
`gostdlib` per application. Gates 03, 04, 07, or 09 must fail with the exact
affected identities.
Replacing per-file language versions with a workspace maximum, comparing Go
versions lexically, or using the catalog maximum as source permission must fail
Gate 02 with the affected occurrence identities.

## Analysis-Scope And Cost Proof

The scope verifier exact-joins every `DefinitionID` to one provider/depth
selection and independently verifies the structural-source plan implied by
every exact-definition, exact-package, namespace, and conditional rule. It
separately joins every source file and definition to identity/type/mapping
evidence; files and packages may aggregate several depths and cannot supply a
default.

Every conditional rule exact-joins its declared finite candidate set and
requested `SelectionFactKind`s to one identity-keyed selection-fact artifact.
The verifier proves scope consumes only those facts, the typed frontend reuses
the same fact IDs, and no second producer or arbitrary predicate callback
exists.

The definition-graph verifier independently derives, by canonical identity:

1. every source-file/synthetic owner region and source-file containment graph;
2. every canonical structural/executable occurrence payload, with the two
   stores disjoint and their exact union equal to all region membership;
3. every definition and its one source/synthetic site, normalized complete
   containment path, and unique rooted-forest parent;
4. every used-once containment anchor and absence of copied occurrence/path
   records;
5. every header membership identity and its canonical parent, edge, role,
   token, ordinal, and exact header byte-range digest;
6. every execution-boundary variant and ordered entry identity/hash;
7. every definition selection;
8. every full-semantic source or implicit executable-region operation;
9. the absence of executable occurrences and reachable body syntax/type keys
   for every non-full definition; and
10. the selected local-or-certified graph authority of every file.

It exact-multiset-joins each class separately with both one-sided identity lists.
A header discriminator, definition count, or shared digest is not structural
evidence.

The combined scope/structure proof establishes:

- workspace and selected source-available module definitions intended for
  automatic translation have recursive checked evidence and conserved
  executable occurrences;
- standard-library definitions used through `gostdlib` retain exact sites,
  headers, and boundaries but contribute no application executable
  occurrences;
- an exact-definition automatic override inside a provider-owned namespace
  upgrades precisely its containing file and succeeds end to end;
- a conditional automatic rule acquires its complete declared candidate set or
  is rejected before loading; no selected full definition reaches a later
  retention mismatch;
- each file's production graph comes from local structural extraction or the
  certified provider artifact, never package-wide full-status selection or a
  merge of two authorities;
- original cgo definitions/sites/headers/boundaries, checked counterparts, and
  external boundaries join without a skipped definition or whole-package
  downgrade;
- cgo joins use toolchain position/origin evidence, include package-synthetic
  definitions, and contain no basename/name-matching fallback;
- a complete cgo package runs through the public pipeline, including source
  verification, structural-source planning, definition graph, mixed-depth
  executable inventory, and semantic evidence lookup; loader-only acceptance is
  not proof;
- Go, non-Go, embed, overlay, and checked-view inputs join by stable typed
  identity rather than raw acquisition path, including after relocation;
- finalized artifacts expose no raw syntax/checker object and no body-indexed
  evidence for non-full definitions;
- catalog-audit occurrences remain audit evidence rather than product
  executable occurrences, and certified provider structures are consumed
  without rescanning non-translated bodies; and
- the provider manifest definition census is built without decoding detailed
  shards, every projected package exact-joins its independently decoded detail
  to manifest membership/census/counts/facts, and the physical cache contains
  zero or one package while the logical graph remains immutable; and
- package closure, structural-source planning, definition depth, header
  occurrences, executable occurrences, and eventual reachability remain
  separate denominators.

The mandatory fixture matrix covers:

- function/method declaration, function literal, multi-name typed package
  initializer, bodyless obligation, and implicit work;
- package `const` and package `var` declarations with the same `ValueSpec`
  shape, proving parent declaration class is part of the catalog query;
- every valid definition-kind/evidence-depth pair and representative invalid
  pairs, including bodyless-as-full and implicit-full-without-operation-graph;
- two same-kind definitions with different headers, proving a kind enum cannot
  stand in for header evidence;
- paired body-only and header-only edits proving the header and execution
  content-address domains do not contaminate one another;
- outer full/child non-full, outer non-full/child full with a captured binding
  declared in the excluded parent, both full, and both non-full;
- three-level nesting, two literals on one line, literals inside package
  initializers, and ordered multi-expression initializers;
- a package containing both locally extracted and certified files;
- provider namespace plus exact-definition automatic override and every
  conditional-acquisition outcome;
- a parent without `C` use and C-dependent child, the inverse, `C` in a literal
  signature, a shadowing local named `C`, and mixed pure/C-dependent definitions
  in one cgo file; and
- missing, duplicate, extra, ambiguous, relocated, and overlaid cgo origins and
  synthetics.

Every mixed-depth artifact is inspected for one definition, one site and
containment path, one header, one boundary, one selection, and the expected
zero/one executable region for both parent and child. The child site remains
present even when its parent executable region is absent. A generic traversal
from a finalized non-full definition cannot reach its body.

Required mutations omit or duplicate an owner/containment graph,
definition/site/header/boundary/selection; orphan, reparent, or cycle a site;
truncate its containment path; copy a shared path prefix into every site; swap a
header parameter, result, `ValueSpec` name/type, edge, role, or order; anchor a
definition to its body entry; include executable bytes in the header digest or
header bytes in an execution digest; drop the child site only in the
outer-non-full/child-full case; restore package-wide provider filtering; ignore
an exact-definition or conditional structural-source rule; merge local and
provider records; expose backing storage or raw syntax; assign depth from
provenance or file state; replace cgo object identity with spelling;
omit/duplicate a cgo origin; compare only synthetic names; duplicate one
occurrence payload across structural and executable stores; or classify a
package `const` `ValueSpec` as an initializer definition. Each fails its owning
gate with exact one-sided identities.

Provider-storage mutations additionally alter a manifest definition identity,
omit or duplicate a requested fact, change package/file membership, change a
shard digest or input digest, make the manifest census disagree with the
independently decoded package detail, construct the census by opening all
shards, retain two package payloads, duplicate detailed records in the
manifest, or replace package-indexed facts/files with repeated global scans.
Integrity/identity mutations fail exact admission or join gates. Residency and
scan mutations fail deterministic production instrumentation and isolated
ordinary-consumption RSS/work budgets; semantic equality alone cannot admit
them.

Construction complexity is verified against the production work ledger, whose
closed operation classes are catalog-edge inspections, boundary/containment-map
lookups/probes, record appends, join/probe operations, and deterministic sort
comparisons. Wide and deeply nested definition fixtures vary nodes,
definitions, unique path anchors, headers, and executable occurrences
independently. Work/storage must follow the architecture's linear equation,
with only named sort work following
`O(n log n)`. A mutation replaces a real constant/logarithmic boundary lookup
with a counted linear scan in the production structural pass; the production
gate—not a separate foil—must fail. A second production mutation materializes
full copied paths per site and must violate the storage/work bound. Wall time
and RSS are corroboration, not the asymptotic proof.

For two ordinary projects and the self-host project, the gate records package,
file, definition, site, header occurrence, boundary, retained executable
occurrence, provider artifact byte, and production-work counts plus elapsed time
and peak RSS. Provider-artifact production and ordinary consuming compilation
are separate rows with separate budgets. Ordinary consumption also records
resident local-detail records, manifest definition identities/requested facts,
largest projected package bytes/records, package-shard loads, and maximum
simultaneously cached packages. It reports the twenty largest header/provider
records. Any material increase without corresponding selected structure and
typed necessity fails before semantic-model work; aggregate improvement cannot
hide a largest-package/provider explosion or eager total-provider residency.

Deterministic identity/work assertions run in-process. Time, peak RSS, and
retained-heap measurements run in isolated subprocesses with exact
toolchain/environment fingerprints, forced lifecycle completion, and repeated
samples; they are not absolute `HeapAlloc` assertions under `-race`. Every
audit/retention set exact-joins expected identities—sampling and `>=` bounds are
not completeness evidence.

Architecture mutations moving structural classification into `internal/source`,
putting source/graph state or policy execution in `internal/scope/contract`,
making `internal/scope/sourceplan` import the definition graph or select depth,
making `internal/language/structure` choose evidence depth, adding an arbitrary
conditional callback or second selection-fact producer, making `internal/scope`
rebuild structural/semantic evidence, making executable inventory interpret
`go/types` or rebuild definitions/headers, making the typed frontend rediscover
roles/selection facts, or retaining raw syntax/checker objects after
finalization must fail. The positive gate proves the exact order
`resolve -> plan structural sources -> definition graph -> selection facts ->
select depth -> executable inventory -> semantic materialization ->
finalization -> independent verification` and non-overlap of all owners.

The finalized cost/attestation gate fails closed in the certification
environment: a portable unit suite may omit the expensive isolated run, but the
phase-exit attestation gate may not silently skip it, and it binds the exact
source, toolchain, provider, environment, and fixture revisions measured.
Every certification command that can launch another compiler or toolchain runs
as one process group with a timeout plus forced-kill grace period, bounded
concurrency, a language-runtime memory limit, and an OS-enforced address-space
or memory ceiling. Output is disk-backed under `.temp/` with a bounded rendered
summary. OOM, timeout, and semantic failure remain distinct retained evidence;
an OOM is never retried uncapped, and a pathological fixture is redesigned
before rerun without weakening the semantic or asymptotic mutation it proves.

Generic context classification also has an independent type-semantics matrix.
It covers normalized unions and intersections, tilde terms, empty and mixed
type sets, named types with equal underlying types, and directional-channel
core-type rules. Replacing the one semantic service with a construct-local
flatten/sort/deduplicate approximation must fail the matrix.
At minimum the matrix includes a nontrivial intersection such as
`interface{ int|string; int|bool }`, whose type set and core type are `int`;
concatenating the two unions is a known-invalid approximation.

## Construct And Semantic-Class Oracles

The typed-frontend verifier exact-joins every `DefinitionID` to one
`DefinitionSemantics` and closed semantic-authority witness, and every retained
occurrence to one legal `OccurrenceResolution`. When checker and certified
provider evidence both exist, it independently compares them and proves only
the contract-selected authority entered the model. Missing, duplicate,
structural-only-without-catalog-disposition, and dual-authority records fail
with exact identities.

Every catalog class has focused source fixtures covering its context matrix.
Oracles compile and execute the same fixture with Go and generated TypeScript
and compare a canonical tagged event stream preserving:

- integer width and exact bigint;
- float bits, NaN, infinity, and signed zero;
- arbitrary string bytes and rune sequence;
- defined/dynamic type, nil, and typed nil;
- values, copies, mutation trace, and storage-equivalence graph;
- map presence, equality, hashing, and allowed range outcomes;
- function/interface targets and assertion diagnostics;
- panic value/category/timing and defer/recover trace;
- initialization and external effects; and
- permitted channel/select schedule outcomes.

The oracle stream is versioned and typed; ordinary JSON coercion and ad hoc
stdout comparison are insufficient.

Mutation tests intentionally break the lowering, plan edge, copy, identity,
hash, dispatch target, or verifier and require the relevant gate to fail. A
test that has never failed under a representative mutation is not accepted as
proof of its claimed defect class.

Method-plan fixtures cover native non-nil calls, nil-panic calls with
effectful arguments, nil-observing pointer receivers, value receivers with
observable copies, exact concrete selection across a planned class hierarchy,
dynamic interface dispatch, promoted methods, generic receiver types, method
expressions, method values, and manual/external bodies. Oracles compare result,
side-effect order, receiver capture, selected MethodID, panic category, and
panic timing. Mutations remove the thunk, move its nil check before argument
evaluation, replace an exact body call with virtual dispatch, remove a value
copy, or generate an unnecessary ordinary-method adapter; semantic or shape
gates must fail.

## Independent Structural Verification

The independent verifier parses final TypeScript with the pinned TypeScript
compiler service and extracts declarations, types, fields, methods, heritage,
generic binders, calls, control forms, imports, ownership markers, and planned
operation annotations. It does not call the emitter's type printer or semantic
canonicalizer.

It compares this structure to Go-side expectations derived from typed source
and the public plan schema. Constants include exact value and type; variables
include storage/type; aliases remain distinct from definitions; structs include
field order, tags, embeds, and visibility; interfaces include normalized method
identity and type sets; callables include receiver, binders, constraints,
parameters, results, and variadic state.

For bodies, it verifies operation conservation and order rather than attempting
to infer Go semantics from arbitrary TypeScript text. Manual bodies are checked
against their generated contracts and execution obligations, not asserted to
be structurally generated.

## Strict Staticness

The complete output is swept as typed AST, including runtime, standard library,
generated code, manual bodies, external adapters, and extensions. The gate
rejects:

- semantic payload recovery from `any`, `unknown`, `object`, `{}`, or aliases
  to them;
- `as any`, `as unknown as`, unchecked assertion recovery, `@ts-ignore`, or
  diagnostic suppression;
- reflection, source-name dispatch, string-selected members, dynamic imports
  used for semantic choice, or host-shape probing;
- invocation through `Function.prototype.call`, `apply`, or `bind`, and
  prototype lookup/manipulation used for method semantics; this is checked by
  resolved symbol identity rather than property-name text;
- erased function invocation or universal operation registries;
- per-call exhaustive interface implementer switches;
- hidden generic operation arguments without a local plan obligation;
- unresolved imports, implicit `any`, CommonJS, or multiple target ABIs; and
- generated definitions or source bytes absent from ownership ledgers.

An explicit external boundary may receive an opaque host value only if its
contract never recovers Go semantics by inspecting that value. Otherwise the
boundary remains unsupported.

## Calibration And Source Shape

Before corpus-scale implementation, a reviewed calibration corpus contains:

- ordinary declarations, calls, methods, control, collections, and generics;
- every accepted representation-exception family;
- interface/object families with small and large implementer sets;
- operation-free and operation-demanding generic examples;
- the largest and highest-expansion real source bodies; and
- manual, standard-library, external, and extension boundaries.

For each fixture, an independent author writes idiomatic strict TypeScript from
the Go source while generated output is hidden. The pair is strict-typechecked
and differentially executed before becoming a baseline.

Shape verdicts are:

- `direct`: generated form matches ordinary hand-written structure;
- `represented`: a local runtime/container/class form is proven necessary;
- `specialized`: source argument shape is preserved through one justified
  specialization; or
- `manual`: exact automatic lowering is unavailable and publication requires
  a completed manual body.

The primary target is approximately `1x` source-shaped output. Numeric budgets
are frozen from the reviewed Go/hand-port corpus before broad lowering is
enabled; they are not chosen from current generated output. Generated ordinary
code must remain within the formatter-level variance of the reviewed hand port.
The expected Go-byte ratio is approximately `1.0-1.35x`, reported separately
from runtime, `gostdlib`, external stubs, manual code, and source maps.

Any ordinary body above `2x` its reviewed hand-port shape, any material p95/p99
growth, or any broad exception category requires stop-the-line architecture
review. A semantic exception may be larger only with typed per-site necessity
and separate attribution; exception bytes cannot be hidden in the ordinary
aggregate.

## Mandatory Cost Metrics

Every materially affected checkpoint records absolute values and parent deltas
for applicable measures:

- Stage-1 owner/containment regions, definitions, sites/unique path anchors,
  header occurrences,
  execution boundaries, selections, executable occurrences, provider/audit
  bytes, provider-production work/RSS/time, ordinary-consumption work/RSS/time,
  provider shard loads, maximum cached packages, identity-census/fact records,
  largest projected package, and twenty largest structural records;
- selected Go and generated bytes, tokens, and AST nodes;
- generated/Go and generated/hand-port ratios;
- per-body median, p90, p95, p99, and maximum expansion;
- twenty largest files, bodies, type expressions, calls, and expansion ratios;
- ordinary/hidden call arguments and receiver-call preservation;
- definitions, references, aliases, adapters, vtables, constructors,
  specializations, checks, carriers, and placeholders;
- duplicated definitions, identity collisions, overwritten artifacts,
  unattributed bytes, and staticness violations;
- runtime, stdlib, external, manual, extension, framing, and source-map bytes
  as separate categories;
- generation wall time and peak RSS;
- strict-TypeScript wall time and peak RSS; and
- representative runtime latency, allocation, and memory.

Always-zero architecture metrics are:

- duplicate semantic definitions;
- artifact identity collisions/overwrites;
- unattributed generated bytes;
- ordinary hidden operation arguments;
- semantic recovery from erased values;
- source-project-specific lowering branches; and
- non-source-shaped sites without typed necessity evidence.

Moving complexity into a helper, type alias, virtual module, cache, or runtime
does not remove cost; that owner is measured separately.

## Mandatory Artifact Review

Reviewers inspect actual final generated artifacts, not only implementation
diffs or reports. Every architecture milestone includes:

- all changed calibration fixtures;
- twenty largest and twenty highest-expansion bodies/types/calls;
- every new representation or exception class;
- all duplicated or newly shared definitions;
- a deterministic ordinary-path sample; and
- affected runtime, standard-library, manual, external, and extension output.

Each review example shows Go source first, then the semantic decision, generated
TypeScript, simpler candidate, size/staticness/runtime impact, and independent
proof. A summary claim or shared hash is not a substitute for opening files.

## Leakage And Deletion Proof

Broad searches and architecture tests prove:

- no acceptance-corpus path/name appears in production logic;
- no old analyzer, IR, emitter, ABI, manual registry, classifier, or fallback
  remains reachable after replacement;
- no duplicate source of identities, semantic kinds, plans, hashes, or graph
  edges exists;
- no body-entry-based `DefinitionID`, enum-only header surrogate,
  depth-filtered definition site, source-owned definition census,
  package-fullness graph switch, or unsupported provider selector survives;
- no raw TypeScript generation occurs outside the target formatter;
- no manual JSON attachment or generated-file preservation exists; and
- no stale test/comment documents a deleted behavior as current.

Representative mutations add a corpus-name branch, omit a catalog variant,
forge a manual hash, remove a dynamic reachability edge, duplicate a
definition, erase an interface payload, and emit an oversized per-target
switch. Package-universe mutations include import-spelling classification,
module-less standard rejection, acquisition/provenance conflation, and omitted
dependency closure. The appropriate gate must fail each mutation.

## Evidence And Attestation

Every machine artifact has a versioned schema, canonical encoding, exact input
fingerprints, and identities rather than counts alone. Conflicting counts are
resolved by sorted identity multiset joins with both one-sided differences.

Evidence names exact denominators: source packages, declarations, bodies,
literals, initializers, occurrences, implementations, modules, artifacts,
manual units, obligations, roots, graph nodes, executed units, and certified
units. Percentages always name numerator and denominator.

An attestation binds the clean implementation revision, input workspace,
toolchain, target compiler, runtime, standard library, contracts, extensions,
tests, reports, and immutable publication root. When stored in version control,
the evidence-only commit's parent is exactly the attested implementation
revision. Stale, ancestor-only, dirty-tree, or non-reproducible evidence is
invalid.

## Stop-The-Line Conditions

Feature work stops and the owning abstraction reopens when:

- the same workaround appears twice;
- a fact is rediscovered by name, text, host shape, or checker re-entry;
- output or compiler state has parallel owners;
- a generated definition repeats or identity is overwritten;
- an ordinary call gains semantic plumbing;
- interface dispatch expands with implementer count;
- output size, typecheck cost, memory, or runtime grows without source growth
  and typed necessity;
- a local exception spreads into ordinary paths;
- an architecture/staticness/performance gate is blocked by the active design;
  or
- final artifacts cannot be fully attributed.

The response is never to raise a threshold, add an allowlist, special-case the
largest corpus site, suppress diagnostics, or call the issue deferred
optimization. The shared truth owner is corrected and the replaced path is
deleted.
