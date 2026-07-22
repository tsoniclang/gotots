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

## What It Means To Validate A Body

Every reachable implementation has an `ImplementationID` and a ledger joining:

- source declaration/body and exact span;
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
| 02 Language catalog | selected Go grammar/tokens/built-ins reconciled; every full-semantic occurrence and every non-full body boundary classified; unknown count zero |
| 03 Scope closure | complete package closure and evidence-depth partition exact-join independent toolchain/environment metadata; every package/root/edge/body has one disposition |
| 04 Identity and census | source/semantic/implementation/artifact multisets join exactly; collisions, overwrites, and unexplained deltas zero |
| 05 Independent parity | independently parsed generated declarations and operation shapes match typed Go source for every class |
| 06 Plan totality | all facts sealed; every selected operation has one validated plan or explicit blocking state; necessity records complete |
| 07 Artifact ownership | implementations, generated AST units, files, definitions, markers, hashes, and provenance join one-to-one |
| 08 Staticness | no erased recovery, dynamic semantic lookup, unsafe casts, unplanned dispatch, or undeclared target operation |
| 09 Regeneration and graph | two clean generations match; manual reconciliation, complete reachability, relocation, and atomic replacement pass |
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

The scope verifier exact-joins every executable body, initializer, and implicit
implementation unit to one evidence depth. It separately joins every source
file and declaration to its identity/type/mapping record; files and packages
may aggregate several depths and cannot supply a default. It proves:

- workspace and selected source-available module bodies intended for automatic
  translation have full checked syntax and conserved occurrence identities;
- standard-library bodies used through the reusable `gostdlib` contract retain
  declaration and body-boundary evidence but contribute no application-body
  occurrences;
- original cgo source, checked syntax, and external boundaries join without a
  skipped file or whole-package downgrade;
- cgo joins use toolchain position/origin evidence, include package-synthetic
  checked declarations, and contain no basename/name-matching fallback;
- Go, non-Go, embed, overlay, and checked-view inputs join by stable typed
  identity rather than raw acquisition path, including after workspace,
  module-cache, and temporary checked-view relocation;
- finalized artifacts expose no syntax or body-indexed `types.Info` for
  declaration-contract, external-boundary, or intrinsic bodies;
- catalog-audit occurrences are stored in audit evidence, never in the product
  semantic artifact, and the exact fingerprinted audit set is consumed without
  rescanning non-translated standard-library bodies per application; and
- resolved-closure, evidence-depth, occurrence, and eventual reachability
  denominators are reported separately.

For two ordinary projects and the self-host project, the gate records package,
file, declaration, body-boundary, and retained-occurrence counts plus elapsed
time and peak RSS. Any material increase without a corresponding change in the
full-semantic source set and typed necessity fails before semantic-model work.
Aggregate package growth cannot hide a largest-package or standard-library-body
scope explosion.

Semantic scope counts are deterministic unit/gate assertions. Time, peak RSS,
and retained-heap measurements run in isolated subprocesses with exact
toolchain/environment fingerprints, forced lifecycle completion, and repeated
samples; they are not absolute `HeapAlloc` assertions embedded in `go test` or
measured under `-race`. Every audit/retention set exact-joins expected
identities—sampling and `>=` bounds are not completeness evidence.

Required mutations analyze a standard-library body as an application body,
drop a source-available dependency body, retain catalog-audit occurrences in
the product model, omit a cgo source/boundary mapping, use a relocated raw path
as an input identity, ignore a cgo-synthetic declaration, restore basename or
declaration-name matching, retain a non-full body AST, assign depth from a file
or provenance default, or merge requested roots with the semantic set. Gates
02, 03, or 16 must fail with exact identities and the cost delta.

Generic context classification also has an independent type-semantics matrix.
It covers normalized unions and intersections, tilde terms, empty and mixed
type sets, named types with equal underlying types, and directional-channel
core-type rules. Replacing the one semantic service with a construct-local
flatten/sort/deduplicate approximation must fail the matrix.

## Construct And Semantic-Class Oracles

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

Every generated-surface checkpoint records absolute values and parent deltas
for:

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
