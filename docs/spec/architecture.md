# Compiler Architecture

## Design Objective

The compiler must be strict by construction. Invalid states are rejected at
their owning boundary instead of surviving as optional fields, string tags,
fallbacks, or emitter guesses. Compile-time complexity is acceptable when it
removes generated complexity; generated complexity is acceptable only for a
local, typed, measured semantic necessity.

GoToTS targets TypeScript, not a particular Go project. Production packages
must depend only on Go language/toolchain contracts, the generic input
workspace, target policy, and explicit environment contracts.

## Required Pipeline

| Phase | Input | Sole output | Forbidden responsibility |
|---|---|---|---|
| Workspace resolution | Go workspace, toolchain, build config | typed package/file/byte universe plus provider-artifact availability | provider choice or target decisions |
| Structural-source planning | resolved universe plus environment/provider contract | validated per-file local-syntax-or-certified-provider plan | physical acquisition facts, evidence-depth choice, or semantics |
| Definition inventory | acquired local syntax plus certified provider graphs | complete depth-independent owner/containment/definition/site/header/boundary graph | evidence-depth choice or semantic typing |
| Selection-fact materialization | definition graph, closed fact requests, transient checker evidence or certified provider facts | immutable identity-keyed facts needed solely by conditional selection rules | provider/depth choice or general semantic lowering |
| Analysis scope | definition graph, selection facts, environment/provider contract | immutable per-definition provider/depth selection | fact rediscovery, provenance defaults, or lowering |
| Executable inventory | definition graph, selection, transient syntax evidence | exhaustive full-semantic executable occurrences and parent-assigned grammatical roles | `go/types` interpretation or target decisions |
| Typed frontend | definition graph, executable occurrences, selection facts, and transient checker/certified semantic evidence | Go semantic model | TypeScript shape |
| Whole-program analysis | complete semantic model plus explicit product/API/test/reflection/extension roots | sealed facts including semantic reachability and root witnesses | emission |
| Planning | semantically reachable model plus sealed facts | immutable total `ProgramPlan` | planning unreachable definitions or source rediscovery |
| TS lowering | semantic model plus `ProgramPlan` | typed TypeScript AST | semantic decisions |
| Baseline formatting | typed TS AST | canonical generated files | manual preservation |
| Completion | new baseline plus prior editable AST | reconciled mixed AST | guessing ownership |
| Implementation reachability | full generated/manual/runtime/stdlib/external/extension graph | retained product graph and root/non-reachability witnesses | text search or semantic-graph substitution |
| Verification | sources, models, plans, artifacts | gate evidence | repairing output |
| Publication | verified staged root | atomic current product | partial replacement |

Each phase consumes a complete typed artifact. A later phase cannot call an
earlier analyzer to fill a missing fact. Errors propagate immediately and do
not create partial records that consumers might interpret as valid.

Workspace resolution identifies selected packages, files, bytes, checker
inputs, and certified provider artifacts without classifying implementation
structure. Scope policy next yields a conservative, validated per-file
structural-source plan. Language structure analysis then derives
the complete depth-independent
owner/containment/definition/site/header/boundary graph from
exactly one local-or-certified source per file. Analysis scope binds every
definition to provider/depth using separately materialized selection facts,
after which executable inventory expands only full-semantic interiors. No later
phase may discover an additional definition.
The catalog audit independently proves language coverage without forcing
provider body retention. This prevents loader provenance, parser availability,
or retention mechanics from becoming implementation policy.

The Stage-1 package seam is strict. `internal/source` owns selected bytes,
physical source-acquisition facts, the single transient AST/checker graph,
source identities, and final severing. It does not classify catalog kinds, bind catalog edges or
roles, resolve lexical tokens, enumerate definitions, or manufacture
occurrence/header/site records. `internal/scope/contract` owns the closed,
versioned provider/rule/selection-fact-request schema and validation, but no
source or definition state. `internal/scope/sourceplan` owns only the
pre-graph structural-source plan and cannot import the definition graph or
choose evidence depth. `internal/language/structure` owns the depth-independent
structural pass and produces the definition graph.
`internal/language/selectionfacts` owns the closed request-driven semantic facts
needed before selection; every such fact is produced once and later reused by
the frontend. `internal/scope` owns only per-definition provider/depth
selection. `internal/language/executable` owns the later parent-directed
structural pass and produces full-semantic occurrences plus grammatical roles,
without interpreting `go/types`. `internal/language/frontend` alone resolves
typed variants, bindings, declaration semantics, and all implicit semantic
operations. Stage 1 catalogs those implicit-operation classes and their
required evidence but produces none of their occurrences. These owners consume
the one catalog authority; none owns a private
edge/variant table or recreates another owner's artifact.

The compiler orchestrates
`resolve -> plan structural sources -> build definition graph -> materialize
selection facts -> select depth -> inventory executable regions -> materialize
semantics -> finalize`.
The structural pass visits each locally acquired syntax node at most once and
provider graph production streams the same responsibility per file. The
executable pass visits each selected full-semantic executable occurrence at
most once. Selection-fact extraction visits only declared finite candidate
definitions and accounts for every probe/edge in the production work ledger.
Purpose-specific bounded passes are preferable to a duplicate source-owned
census or a depth-dependent topology pass; no pass may recreate another's
artifact.

No raw `ast.Node`, `types.Package`, mutable `types.Info`, `types.Scope`,
`types.Object`, `token.FileSet`, selection, or expression can survive in the
finalized Stage-1 API; downstream access is by canonical identity through
narrow read-only facts. Finalization actively severs transient syntax and
checker access: the finalized artifact holds no field and exposes no accessor
that reaches a `go/ast`, `go/types`, or mutable `go/token` object, and transient
graph access after finalization is structurally impossible, not merely
discouraged.

Stage 1 owns syntax inventory, owner regions, definition identities and sites,
header regions, execution boundaries, scope/acquisition selection, and
finalized structural artifacts. Stage 2 consumes those exact records and reads the same transient
checker graph before finalization to materialize target-independent declaration
semantics, bindings, objects, captures, types, and executable operations. It
does not rebuild loading, identities, Stage-1 visitors, or structural regions,
and it never reintroduces a finalized checker.

The transient checker graph's lifetime is owned by `internal/source` and is
defined exactly. The one checker graph is live from workspace load through
finalization. Selection-fact materialization reads it through a narrow
request-bound view; Stage 2 consumes that same graph and the already-materialized
selection facts, then materializes its canonical, identity-keyed semantic facts
BEFORE source finalization severs it. There is no finalized raw type-fact API
and no second checker: the typed frontend reads the transient graph in place and
emits identity-keyed facts. The inspect-only
pipeline ends after Stage 1, so it may finalize immediately, with no semantic
materialization step. A finalized artifact never carries a mutable checker
object for a later phase to consult. An occurrence of an identifier is not, by
itself, proof of a resolved binding, capture, or object identity; captured-
binding and object facts are materialized by the semantic model from the
checker graph, never inferred from an occurrence's presence.

## Input Contract

A compilation request contains:

- workspace/module roots and overlay files;
- selected Go toolchain identity and `GOOS`, `GOARCH`, tags, experiments, and
  relevant environment;
- executable, API, test, and extension roots;
- source inclusion/exclusion rules with provenance;
- target TypeScript/runtime policy;
- standard-library version and implementation root;
- external contracts; and
- completion/publication mode.

The request schema is closed and versioned. Unknown fields and ambiguous root
rules fail. Project configuration may select source and environment contracts;
it cannot register new lowering semantics or branch on a package name.

All input files, tool versions, configurations, external contracts, extension
packages, and prior generated baselines used for reconciliation are
fingerprinted before compilation. Machine-local paths never enter semantic
identity.

Every Go, non-Go, embed, overlay, and checked-view input has a constructor-
validated owner-relative identity distinct from its acquisition/display path.
Generated cgo checked files are identified by their exact source-to-checked
relation, never by a temporary-directory spelling. Raw absolute paths may be
used to read bytes during loading, but cannot be join keys or survive in a
portable semantic artifact.

## Resolved Package Universe

The workspace loader resolves the complete transitive package closure under the
exact selected Go binary, workspace, build flags, environment, tags, and
overlays. Every package has one constructor-validated record whose independent
facts are:

| Fact | Closed values | Authority |
|---|---|---|
| package owner | `module(ModuleID)`, `standard-library`, `toolchain`, `language-pseudo` | semantic identity domain |
| provenance | `workspace-module`, `module-dependency`, `standard-library`, `toolchain-package`, `language-pseudo` | resolved `go list` package/module metadata and pattern membership |
| acquisition | workspace, module cache, vendor, local replacement, `GOROOT` | resolved build inputs |
| language disposition | ordinary source, built-in universe, `unsafe` intrinsic, cgo pseudo-package | selected Go language/toolchain contract |

Module-backed package identity contains the declared module path and selected
module version, not its checkout, cache, vendor, or replacement directory.
Standard-library and other toolchain packages do not have Go module metadata;
they use reserved typed owner identities rather than a fabricated module or a
`GOROOT` path. The exact toolchain and standard-library contents remain bound by
input fingerprints and contract hashes, so relocation preserves identity while
a toolchain upgrade produces explicit structural deltas.

Provenance is derived only from the selected Go command's resolved metadata and
authoritative pattern sets. Standard-library provenance means membership in
`go list std` under the exact build request; `Standard` and `Goroot` remain
supporting facts but cannot alone distinguish the standard set from toolchain
commands. Other inputs include `Module.Main`, module version/replacement,
vendor selection, and `go list cmd` membership where applicable. Rules based
on a dotless import path, path prefix, `Module == nil`, or physical containment
under an assumed `GOROOT` are forbidden. A local replacement remains a module
dependency with local-replacement acquisition; an overlay changes selected
bytes and fingerprints, not package identity or provenance.

`builtin` contributes the predeclared universe and emits no package module.
`unsafe` is a standard package whose intrinsic operations receive explicit
semantic dispositions. Import `"C"` is a cgo pseudo-package whose obligations
must be planned explicitly. Neither special case turns an importing package
into an external package.

Go semantics are uniform across provenance, but resolving a package does not
authorize retained analysis of every executable interior. Stage 1 treats
implementation structure and executable evidence as orthogonal:

```text
depth-independent definition graph
    = owner region
    + normalized sparse containment graph
    + implementation definition
    + exactly one definition site
    + exactly one header region
    + exactly one execution boundary

request-bound scope overlay
    = exactly one definition selection per implementation definition

depth-dependent executable evidence
    = exactly one executable region only when depth is full-semantic
```

An **implementation definition** is one function or method declaration,
function literal, package-level `ValueSpec` with initializer expressions,
bodyless implementation obligation, or cataloged implicit implementation with
independent provider, depth, and implementation ownership. Ordinary implicit
semantics inside another definition—copying, zeroing, boxing, promotion,
conversion, dispatch, and similar operations—remain semantic operations within
that definition and never become definitions. Local initializers likewise remain
executable occurrences in their enclosing definition; only package-level
`ValueSpec` initializers own definitions.
A bodyless obligation is a bodyless `FuncDecl` or an exact environment contract
that requires a concrete implementation. Interface method fields and function
types are declaration contracts, not implementation definitions.

Its `DefinitionID` is anchored to the construct root or typed implicit owner.
For a source definition it contains the canonical `FileID` and construct-root
occurrence identity, so an exact selector identifies the containing file
without loading the body. It never reuses a body block, initializer expression,
or other execution entry as the definition identity. It is revision-bound
source evidence, distinct from the later `ImplementationID` of a concrete
planned/emitted specialization.

Every definition receives one closed evidence depth through its separate
`DefinitionSelection`:

| Evidence depth | Retained evidence | Typical members |
|---|---|---|
| `full-semantic` | definition, site, header, execution boundary, checked contextual executable region | workspace and source-available module definitions selected for automatic translation |
| `declaration-contract` | definition, site, complete header, execution boundary spans/hashes, no executable occurrences | standard-library behavior supplied by `gostdlib` and other manual environment contracts |
| `external-boundary` | definition, site, complete available header, execution boundary, checked-view mapping where available, typed unresolved obligation | exact cgo/native/host-owned definitions |
| `intrinsic` | typed definition/site/header owner and intrinsic operation descriptor, no ordinary source executable region | `builtin`, `unsafe`, and genuine language/toolchain intrinsics |

Definition kind and evidence depth form a validated closed compatibility
matrix:

| Definition class | Ordinary-source depths | Intrinsic-package depths |
|---|---|---|
| bodyful function/method declaration, function literal, package initializer | `full-semantic`, `declaration-contract`, or `external-boundary` | `intrinsic` only |
| bodyless implementation obligation | `declaration-contract` or `external-boundary` | `intrinsic` only |
| cataloged implicit implementation | `full-semantic`, `declaration-contract`, or `external-boundary` | `intrinsic` only |
| package-synthetic checked/external adapter | `external-boundary` only | none |

Invalid dispositions and depths reject every definition. An intrinsic package
cannot be rebound to a non-intrinsic provider, and an ordinary package cannot be
rebound to intrinsic depth. A source declaration/literal/initializer may be
full only with complete recursive checked evidence. A cataloged implicit
implementation may be full only with an exact typed implicit executable graph.
A bodyless obligation cannot be full-semantic merely because a provider exists;
it remains a declaration/external boundary until a later concrete manual or
external `ImplementationID` satisfies it. Every invalid pair fails selection.

Files and packages may aggregate definitions at several depths and therefore
have no inferred depth. The four sets are disjoint and total by `DefinitionID`.
Package provenance and a file-wide transformed/cgo bit are inputs, never the
policy. No function may assign depth from provenance, package, or file state
alone even when one profile happens to select every definition alike.

The compilation request selects a provider contract and certified artifacts and
records their canonical digests. Before definition inventory, the compiler
derives structural-source requirements from **every** exact-definition,
package, namespace, and conditional rule:

- an exact-definition rule identifies its containing file directly;
- a package or namespace rule contributes all matching files;
- a conditional rule declares the evidence it consumes and a sound finite
  candidate file/definition set supplied by its selector or certified graph;
  its predicate consumes only closed `SelectionFactKind`s owned by
  `internal/language/selectionfacts`, never an arbitrary callback; if
  any candidate can become `full-semantic`, its containing file is recursively
  available; and
- a rule whose full-semantic candidate set cannot be determined without the
  evidence it is supposed to acquire is invalid and fails contract validation.

The structural-source plan is the union of those requirements. It neither
silently widens every provider package nor ignores a narrower override.

Definition inventory then produces the complete pre-scope definition graph.
Its definitions are the authoritative **definition census**; any flat census is
a derived projection. Closed, permanently pinned definition-kind IDs cover all
five definition forms above. Each source definition carries its `DefinitionID`,
kind, file, construct-root anchor, header, and execution-boundary variant; an
implicit definition carries its typed semantic owner and catalog operation,
never a fabricated source span or display name. Local structural traversal or a
content-addressed provider graph supplies each file. Executable inventory may
link occurrences to these definitions; it may never create one.

Before scope binding, the selection-fact owner materializes exactly the
requested facts for the declared finite candidate set from the one checker
graph or a certified provider fact artifact. Every fact has a closed kind,
canonical `SelectionFactID`, typed payload, producer/evidence digest, and exact
consumer rule set. Facts are immutable and become input to Stage 2; the
frontend may consume but never rediscover them.

Analysis scope finally binds every censused definition. Contract resolution is
total and evidence-producing: an exact-definition binding, exact-package
binding, declared owner-namespace rule, or declared conditional rule identifies
the provider and depth of every definition. Broad rules are contract data, not
compiled provenance defaults. The selection ledger records the contract digest,
selected provider/depth, rule identity, and exact selection-fact witnesses for
every definition. A selected full-semantic definition lacking recursive
source/checker evidence fails before executable inventory; a valid advertised
rule cannot fail later as a retention mismatch.

The depth-independent structural graph and separate scope overlay contain these
exact records:

| Record | Required payload and cardinality |
|---|---|
| `OccurrenceStore` | one canonical immutable payload per retained `OccurrenceID`: kind, actual grammatical parent, catalog edge, parent-assigned role, ordinal, physical span, display span, and lexical token evidence; Stage-1 structural and executable stores are disjoint and exact-union to this one namespace |
| `OwnerRegion` | exactly one closed `SourceFileRegion` per selected source file or typed `SyntheticOwnerRegion` per canonical synthetic semantic owner; a source region references every ordinary structural occurrence outside implementation definitions and its one containment graph, but never copies occurrence payload |
| `ContainmentGraph` | exactly one immutable parent-linked sparse graph per source-file region; it stores only the canonical occurrence identities needed to connect source definition sites to their nearest owner/boundary, and contains no copied occurrence facts or unrelated excluded-body occurrence |
| `ImplementationDefinition` | one `DefinitionID`, pinned kind, owning source file or synthetic owner, `HeaderRegionID`, and `ExecutionBoundaryID`; no provider or evidence-depth field |
| `DefinitionSite` | exactly one closed `SourceContainmentSite` or `SyntheticOwnerSite`; a source site carries its owning region/definition and terminal construct-root occurrence identity, whose parent chain through the canonical occurrence store and containment-anchor identity set is complete and rooted |
| `HeaderRegion` | exactly one immutable ordered list of canonical occurrence identities rooted at the definition and containing every cataloged non-executable edge, with no executable-entry or nested-definition interior |
| `ExecutionBoundary` | exactly one closed variant: block entry, ordered initializer-expression entries, bodyless obligation, or typed implicit operation; source entries reference canonical occurrence identities and carry only their independent entry content hashes |
| `DefinitionSelection` | exactly one per `DefinitionID`, in the scope overlay rather than the structural graph: provider, evidence depth, contract/rule identity, and evidence witness |
| `ExecutableRegion` | zero or one closed `SourceExecutableRegion` or `ImplicitExecutableRegion`; required exactly for every full-semantic definition and forbidden for every non-full definition; source regions contain ordered occurrence-identity membership and definition references, while their additional occurrence store is disjoint from the structural store |

Header and execution content addresses are independent. A header digest covers
only canonical header kinds/edges/roles/order/tokens and exact header byte
ranges;
it excludes executable-entry bytes and any diagnostic full-construct extent
whose only change comes from executable content. Each source execution entry
has its own digest, and ordered multi-entry boundaries have a separate combined
digest. A body-only edit therefore changes execution evidence without
masquerading as a header change.

Header byte ranges are construct-defined and closed: a function declaration or
literal uses its exact prefix ending immediately before the body block; a
bodyless declaration uses its complete declaration; and a package initializer
uses the `ValueSpec` prefix ending immediately before its first value. Expression
bytes, separators between values, and nested-definition bodies are not smuggled
into a header digest through a full-root span. Header occurrence membership and
header byte ranges are independently joined.

`DefinitionSite` is structural containment, not a general semantic reference.
For source definitions the referenced containment path begins at the nearest
owner region or enclosing definition boundary and ends at the construct root.
Paths are normalized: shared prefixes and anchors are stored once and recovered
through the canonical occurrence store's parent links, never copied into each
site. The containment graph records only anchor membership. The same path
exists at every evidence depth. When an anchor is also a full executable
occurrence, both relations reference the same canonical payload. No kind,
parent, edge, role, order, token, span, or content fact is duplicated.
Later binding, call, dispatch, initialization, and reachability references are
separate relations and may repeat. A package initializer's source site remains
the `ValueSpec`'s grammatical `GenDecl.Specs` location; package-initialization
ordering is a separate typed operation edge and never replaces that site.
When a cataloged implicit package-initialization coordinator is represented as
its own definition, it owns only ordering and invocation edges. Each
`ValueSpec` definition remains the sole owner of its initializer evaluation and
stores.

The construct partition is generic and closed:

| Definition form | Header region | Execution boundary |
|---|---|---|
| function/method declaration | `FuncDecl` root plus docs/directives, receiver, name, type parameters, parameters, and results; excludes `Body` | one `Body` block |
| function literal | `FuncLit` root plus its function type, parameters, and results; excludes `Body` | one `Body` block |
| package initializer | `ValueSpec` root plus docs/comments, names, and optional declared type; excludes `Values` | ordered `Values` expression roots, preserving arity and order |
| bodyless declaration | complete declaration/signature header | typed bodyless obligation; no source executable entries |
| implicit implementation | exact catalog/type owner descriptor with independent implementation ownership | typed implicit execution boundary and, when full-semantic, typed implicit executable graph |

This table is derived from the catalog's executable-entry classification, not a
second hand-maintained AST edge list. A new Go construct cannot enter the model
until the catalog classifies every child edge as header, execution entry,
nested definition site, or ordinary executable child. Unknown classification
fails before artifact construction.

Definition classification is parent-directed and context-complete. Its closed
query includes lexical scope, parent construct/edge, and any declaration token
or class needed to distinguish equal child node shapes. For example, these two
`ValueSpec` nodes are not the same implementation class:

```go
const Answer = 42 // compile-time declaration; no package-initializer definition
var Answer = 42   // package initializer; one implementation definition
```

The parent `GenDecl` supplies `const` versus `var` to the catalog. The
`ValueSpec` never inspects its parent or source text, and a classifier that uses
only `KindValueSpec + hasValues` is invalid.

For example:

```go
var Transform = func(x int) int {
    step := func(y int) int { return y + 1 }
    return step(x)
}
```

Stage 1 records three definitions: the `ValueSpec` initializer, the outer
`FuncLit`, and the inner `FuncLit`. The outer literal's site path begins at the
initializer boundary's first `Values` entry. The inner literal's site path
begins at the outer literal's block boundary and ends through the assignment's
right-hand edge. The local binding `step` is not another implementation
definition. If the outer literal is declaration-contract while an exact rule
selects the inner literal full-semantic, all three definitions/sites/headers/
boundaries remain, the outer literal has no executable region, and the inner
literal has exactly one. No excluded parent AST is needed to find or analyze
the selected child.

Occurrence ownership is singular. A source-file owner region owns ordinary
non-definition structure and stops at each definition site. Every definition
header owns its non-executable occurrences. A full executable region owns only
the occurrences reachable from its execution entries and stops at nested
definition sites. Header occurrences and executable occurrences never overlap
or duplicate records. Sparse-containment anchors are relation records, not a
second occurrence payload. A site inside an excluded parent executable region
remains independently retained; its validity never depends on the parent's
executable region being materialized.

The owner-region/site relation forms a rooted acyclic forest: every source
definition is reachable exactly once from its source-file region, every
synthetic definition exactly once from its synthetic owner, and every nested
definition exactly once through its immediate enclosing definition. No orphan,
second parent, containment cycle, or span-derived parent is permitted.

For each file the production definition graph has exactly one authority:
locally extracted recursive source or the independently certified provider
artifact. A local acquisition upgrade may use the provider artifact as
corroboration, but the two graphs are exact-joined and only one enters the
finalized inventory. Selection is by file/definition identity, never by whether
a package has any full-semantic definition. Mixed packages and mixed files are
ordinary.

The exact object, binder, type, and capture evidence consumed by Stage 2 remains
in the one transient checker graph until semantic materialization. Stage 1
retains syntax structure only; a header region is not a semantic signature.
Stage 2 produces the canonical definition semantics and binding facts before
finalization severs checker access. No Stage-1 enum or string label may stand in
for either the header occurrence graph or Stage-2 semantic facts.

An excluded executable interior is unreachable through the finalized API. All
definitions use the same owner/containment/definition/site/header/boundary API
at every depth; no
raw parent syntax plus boundary list, respect-boundaries flag,
consumer-controlled skipping, exported raw-node slice, or uniform-full/mixed
consumer split exists. Internal storage optimizations are permitted only when
the public artifact and all consumers remain identical.

Finalization and independent verification enforce these exact multiset joins:

1. selected source files and synthetic owners to owner regions, exactly
   one-to-one;
2. source-file regions to normalized containment graphs, exactly one-to-one,
   with every source site resolving one complete path and every retained anchor
   used;
3. pre-scope census identities to implementation definitions;
4. definitions to definition sites, exactly one-to-one, and all sites to one
   rooted acyclic containment forest;
5. definitions to header regions, exactly one-to-one;
6. catalog-derived header occurrences to header-region occurrences;
7. definitions to execution boundaries, exactly one-to-one;
8. definitions to definition selections, exactly one-to-one;
9. full definitions to executable regions and their exact source or typed
   implicit operation sets;
10. non-full definitions to zero executable occurrences and no reachable body
   syntax/type-information key; and
11. local or certified provider graphs to the one finalized graph.

No join filters owner regions, sites, headers, or boundaries by evidence depth.
Spans and hashes prove identity/content but never rediscover topology. A
provider artifact carries one complete **logical**
owner/containment/definition/site/header/boundary graph, physically partitioned
into independently content-addressed package shards. Its small resident
manifest carries only package/file membership, the exact sorted
`DefinitionID` census, aggregate header/boundary cardinalities, requested
selection facts, and shard digests/offsets. Those records are identity and
admission projections, not substitute semantic payloads. Detailed owner,
occurrence, containment, site, header, boundary, and checked-view records occur
once in their owning package shard and are not duplicated in the manifest.

At provider-artifact production each package shard is independently extracted
and certified before publication. At ordinary consumption the externally
selected container digest admits the manifest; the manifest census is sealed
into the whole-universe definition census without opening every shard; and a
consumer projects at most one detailed provider package at a time. Projection
exact-joins the decoded shard to its manifest membership, definition census,
header/boundary counts, input digest, and requested facts before exposing it.
The hidden physical cache may retain at most one package and cannot alter the
immutable logical API. No application-wide provider-detail map or slice exists.
A flat definition list is legal only as this reconstructible identity-only
census.

Ordinary consumption therefore trusts the request-bound certified digest plus
its own census/selection joins and does not rescan provider interiors.
Ordinary structural verification never opens a provider-only package shard:
the independently certified manifest is the authority for that package's
identity census. It opens one shard only when a selected local overlay must be
exact-joined to certified detail or when a downstream consumer explicitly
requests that package's detail. Sequentially opening every shard merely to
reverify already-certified interiors is the same forbidden whole-provider
work class as retaining every shard at once.
Graph ownership stays in `internal/language/structure`; source owns selected
bytes, physical source-acquisition facts, and transient evidence lifetime and
never imports the catalog to build or census definitions.

Construction has an explicit cost model. The production work ledger counts
every scalable operation class: catalog edge inspection, definition-boundary
lookup/probe, record append, identity join/probe, and deterministic sort
comparison. Total work and storage are
`O(nodes + edges + definitions + sites + unique containment anchors + header
occurrences + retained executable occurrences)`, plus `O(n log n)` only for
named deterministic sorts.
A per-node counter that ignores work performed inside the visit is not evidence.
No definition performs a linear scan over all definitions, boundaries, or
occurrences; quadratic and cubic construction are forbidden.

Provider production and ordinary consumption have separate cost equations and
budgets. Production may stream every selected package once and writes
`O(total provider detail)` bytes. Ordinary resident storage is
`O(local detailed graph + provider definition identities + requested manifest
facts + largest projected provider package)`, never `O(total provider detail)`.
Ordinary work is linear in the resident projections plus each package detail
actually visited; it may not repeatedly scan the manifest or all package facts
per file/package. Provider-production wall/RSS/artifact bytes and
ordinary-consumption wall/RSS are reported separately. Loading every package
shard to construct the census, retaining more than one projected package, or
duplicating detail into the manifest is a stop-the-line regression even if
semantic joins still pass.

Requested roots, resolved import closure, full-semantic source set,
declaration-contract set, and later product reachability are different sets with
different identities and denominators. Root status is only a reachability
witness. Package provenance alone does not choose a TypeScript representation,
but the selected environment contract may bound analysis depth when a Go body
is not an implementation candidate.

All depths share one coherent `go/types` universe. Source-available module
dependencies selected for automatic translation receive full-semantic evidence.
During application compilation, standard-library declarations participate in
typing but standard-library Go body interiors do not enter the application
occurrence model; generating or upgrading the reusable `gostdlib` workspace is
a separate versioned compilation. Original cgo source and its checked view are
kept as distinct, joined artifacts: unaffected checked bodies may remain
full-semantic, while an exact C-dependent body becomes an external boundary.
No source file or body disappears because its checked view differs.

Cgo correspondence comes from selected-toolchain output facts and source
position/line-directive evidence. Basename conventions and declaration-name
matching are not semantic joins. Each source definition, site, header, and
execution boundary maps to its checked counterpart by selected-toolchain origin
evidence with exact kind agreement and exact-one cardinality; when column
evidence is absent, a same-line relation is valid only if the candidate set is
exactly one and otherwise fails as ambiguous. Package-synthetic checked
declarations such as cgo-generated types and call adapters receive explicit
typed `DefinitionID`s, implicit-owner sites, headers, and external/intrinsic
boundaries; they are never ignored as unmatched extras, and identity joins
compare the complete package/name/role tuple, never the name alone.
C-dependence is resolved from checked semantic/toolchain object identity, not
from an identifier spelled `C`, a checked temporary filename, or another
source-text heuristic. It is computed over one definition's header and
execution boundary/region and is never inherited merely because a nested
definition uses `C`.

Ownership is singular: `internal/source` records raw selected-toolchain
source/checked/origin evidence without classifying it;
`internal/language/structure` builds the exact structural correspondence;
`internal/language/selectionfacts` produces the closed per-definition
C-dependence fact; and `internal/scope` only consumes that fact. None may repeat
another layer's derivation.

The independent structural-origin verifier is a separately implemented
extractor. It does not invoke the producer's critical enumeration or semantic
helpers; a cross-check that shares the producer's origin enumeration, synthetic
classification, or object-use walk is an invariant check, not an independent
derivation. Structural origins are verified by that separate extractor, and
semantic C-dependence is proven through selected-Go fixtures and mutations.

Syntax and body-indexed `types.Info` for non-full depths may exist transiently
inside the authoritative semantic load. After definition sites, headers,
execution boundaries, semantic facts, and checked mappings are extracted, the
finalized source artifact severs those references and retains only immutable
identity-keyed artifacts permitted by the selected depth. Loader lifetime
cannot silently turn resolution evidence into retained application semantics.

This lifecycle is structural: transient loaded evidence and finalized source
evidence are different validated types. A mutable `syntax = nil` plus a
`severed` flag, or an unfiltered package-wide `types.Info` retained because one
body is full-semantic, is a forbidden dual state. Mixed files expose only the
per-definition syntax/type evidence their depths permit.

Finalized artifacts are immutable by observable capability, not by convention.
Owned slices and maps return isolated values; no API can reach excluded syntax;
mutable toolchain objects such as scopes and expressions are hidden behind
narrow read-only query views or an enforced internal capability wall; mutation
methods are forbidden outside the authoritative owner; and scopes, initialization
order, file versions, implicits, selections, generic instances, definitions,
uses, and types all come from one definition/executable-region-filtered evidence
owner. Initialization evidence references canonical definitions and occurrences
rather than leaking excluded raw expressions. A shallow reflection check over
seeded accessors is supporting
evidence, not proof of transitive isolation: the immutability gate must also
prove no finalized path reaches an excluded body, that nested collections and
evidence variants expose no backing storage, and that narrow views expose no
mutable toolchain object. Body-indexed type information is selected by exact
retained region identity, never by unqualified byte offsets that can collide
between files. A mutable `*types.Info` map is not a downstream artifact API.

Catalog-coverage scans over the complete Go toolchain or a large corpus are
separate streaming verification workloads. They may prove catalog coverage but
must not enlarge a normal compilation's retained semantic model. Resolution
closure, catalog-audit closure, semantic-analysis scope, and implementation
reachability must never be conflated.

The complete-toolchain audit is a versioned, fingerprinted gate/command run for
the selected Go contract, build configuration, and contract upgrades. Ordinary
application compilation verifies and consumes that audit artifact; it does not
reparse and walk every non-translated standard-library body on every
invocation. Audit membership is an exact identity set, not a lower-bound count.
Each audit record binds its `FileID` to the selected-byte digest and its exact
occurrence/directive evidence. The artifact also binds the provider contract,
catalog structure, selected toolchain binary/source contract, overlays, and all
build-selection inputs such as GOOS, GOARCH, experiments, tags, and build flags.
Verification rejects duplicate records, incorrect per-file or aggregate counts,
changed bytes under an unchanged identity, and any one-sided membership delta.
These comparisons use input hashes already captured by source resolution, so
ordinary compilation does not rescan audited bodies.

An artifact's self-digest proves integrity only: a tampered-then-resealed
artifact carries a valid self-digest. Ordinary consumption gains authority from
an independently selected certified digest bound on the request, never from the
artifact's own seal, so a resealed omission or flipped evidence field fails
consumption. Audit certification binds the overlay and build-input projection
that can change audited membership or audited selected bytes; per-file
selected-byte joins carry the overlay proof, so an application-only overlay that
does not change any audited file's selected bytes must not invalidate a reusable
toolchain artifact.

The compiler-supported catalog version, selected toolchain version, each
module's `go` directive, and the effective language version of each source file
are separate facts. Every occurrence uses the effective file version reported
by typed toolchain evidence, including file-specific version effects. A
workspace-wide maximum, lexical version comparison, or catalog maximum cannot
admit a construct in a file governed by an older language version.

## Complete Go Construct Catalog

The selected Go version has one machine-readable language registry composed of
separate closed catalogs, not one Cartesian-product enum:

1. syntax/node kinds;
2. parent-to-child edges, grammatical roles, and traversal order;
3. lexical tokens and token predicates;
4. context-dependent semantic variants exposed by `go/types.Info`;
5. predeclared identifiers and built-ins;
6. directives and versioned language features; and
7. implicit operations such as zeroing, copying, receiver adjustment, method
   promotion, interface conversion, initialization, and panic boundaries.

Each fact has one domain owner. An occurrence exact-joins the applicable domain
records by identity; no domain copies another domain's values and no combined
mega-kind becomes a second authority. The registry is authoritative Go code,
not parallel JSON maintained by hand. Generated reports may render it. Tests
reconcile each domain independently against the selected toolchain's concrete
AST node types and node-bearing fields, token set/predicates, built-in universe,
directives, and Go language-version features.

Every entry has an explicit permanently pinned ID, descriptive name, version/
disposition metadata, and domain-appropriate evidence. Syntax and edge entries
define allowed roles and traversal; semantic variants and implicit operations
define required typed evidence and produced semantic outcomes; all have focused
positive, negative, and mutation fixtures.

Terminal sentinels, exact-size tables, and bidirectional reconciliation make an
omitted or extra entry fail. There is no default `other`, textual prefix
classifier, unvisited exported node-bearing field, private traversal table, or
generic recursive fallback.

## Context-Aware Construct Analysis

The typed-frontend input for one occurrence is:

```text
syntax node
  + parent-assigned grammatical role
  + expected type and arity
  + lexical/storage/control environment
  + exact go/types evidence
```

Stage 1 uses controlled recursive descent to assign grammatical roles from
parents to children. Stage 2 dispatches each retained occurrence to one
construct-family semantic resolver using that recorded role and the one checker
graph. A resolver must not inspect its parent, source spelling, or emitted
context to infer meaning.

Required role families include:

- value, condition, return value, and discarded value;
- assignable place, addressable place, and declaration binding;
- callee, ordinary argument, variadic argument, and deferred argument;
- type expression, conversion target, composite-literal element, and type
  constraint;
- one-result and multi-result contexts;
- loop, switch, select, deferred-call, recovery, and branch targets; and
- package, function, closure, and initialization scopes.

### Context Example: Map Index

```go
value := values[key]
value, ok := values[key]
```

The first assignment gives the index expression a one-result value role. The
second gives it a two-result comma-ok role. `go/types.TypeAndValue.HasOk()`
confirms the second semantic variant. Analysis produces different operations:

```text
MapLookup(map, key, wantPresence=false)
MapLookup(map, key, wantPresence=true)
```

Planning may then produce:

```ts
const value = values.get(key);
const [value, ok] = values.lookup(key);
```

The index visitor does not scan its parent assignment and the emitter does not
rediscover result arity.

### Context Example: Call Or Conversion

```go
result := F(value)
result := T(value)
```

Identical syntax can mean a call or conversion. Selection comes from the
resolved `go/types.Object` and `TypeAndValue`, producing `Call` or `Convert`.
Name capitalization, argument count, and source text have no authority.

Operations defined over type sets—core type, structural terms, assignability,
method sets, comparability, and constraint satisfaction—belong to one exact
toolchain-semantic service. Construct visitors consume its typed result. They
must not embed local approximations of union flattening, interface intersection,
tilde terms, or channel-direction rules merely to classify a newly encountered
generic construct. The service is differential-tested against accepted and
rejected programs under the selected Go toolchain.

## Target-Independent Semantic Model

The semantic model is the smallest common representation that prevents each
lowering from reimplementing Go rules. It contains immutable typed records for:

- packages, declarations, bindings, fields, methods, and initializers;
- semantic types, aliases, defined types, binders, constraints, and method
  sets;
- values, storage places, loads, stores, address-taking, and copies;
- calls, conversions, interface operations, and generic instantiations;
- evaluation sequence, branches, loops, switches, range, `defer`, panic, and
  recovery;
- closures, captures, function/method values, and multiple results; and
- channels, sends, receives, `select`, goroutines, and synchronization effects.

Each record has a validating constructor. Required fields are not pointers or
zero values standing for unknown. Closed variants use enums plus total
validation and exhaustive switches. Records carry canonical IDs and source
spans, never persisted pointer addresses.

The model states Go behavior, not target mechanisms. It may say
`CopyValue(TypeID)`, `InterfaceConvert`, or `MapLookup`; it cannot say
`emitClass`, `useBigInt`, `callHelper`, or contain TypeScript text.

Every Stage-1 definition has exactly one target-independent
`DefinitionSemantics` record keyed by `DefinitionID`. It materializes the
definition's receiver/signature, declared names and types, bodyless obligation,
initializer contract, or implicit-operation meaning from exactly one semantic
authority: the coherent transient checker graph or an independently certified
provider semantic contract. Full-semantic definitions additionally materialize
semantic operations for their source or typed implicit executable region. A
header region is syntax evidence and cannot substitute for
`DefinitionSemantics`; conversely Stage 2 does not rebuild header topology.
Each record carries a closed `CheckerAuthority` or
`CertifiedProviderAuthority` witness and its input digest. When both sources are
available they may be exact-joined as corroboration, but only the authority
selected by the contract enters the semantic model.

Every retained Stage-1 occurrence has exactly one semantic resolution domain:
`owner`, `header`, `boundary`, or `executable`. Region membership is not copied
into the occurrence payload. If one canonical occurrence is referenced by more
than one Stage-1 relation, its resolution domain is selected by the closed
precedence `executable > boundary > header > owner`: a full definition's entry
is executable, the same entry of a non-full definition is a boundary, and
header/owner syntax never masquerades as an executable operation. Every such
occurrence has exactly one `OccurrenceResolution`. Its closed variants are
`StructuralOnly(DispositionID)`, `DefinitionComponent(DefinitionID,
ComponentKind)`, `Declaration(SemanticDeclarationID)`,
`Binding(SemanticBindingID)`, `Type(SemanticTypeID)`,
`Operation(OperationID)`, and `Unsupported(UnsupportedID)`. The catalog declares
which variants are legal for each kind/role/semantic variant and resolution
domain. `Operation` is legal only in the executable domain. A non-executable
constant/type expression may be `StructuralCompileTimeExpression` only when its
structural payload names exactly one covering `SemanticDeclarationID` or
`SemanticTypeID` whose canonical record conserves the expression's meaning;
this is positive typed evidence, never a generic fallback for an unclassified
expression. Other structural dispositions carry no coverage target. Boundary
entries resolve as exact definition components or explicit unsupported
records. Semantic records may refer to other semantic IDs, but they cannot
silently consume an occurrence owned by another resolution or leave an
occurrence unresolved.

`DefinitionSemantics` preserves declaration multiplicity and order rather than
pretending every implementation definition owns one name:

| Stage-1 definition class | Semantic form | Owned declaration IDs |
|---|---|---:|
| ordinary function or method declaration | callable | exactly one |
| blank function declaration (`func _()`) | callable | zero; it has a signature and body but introduces no package declaration |
| package initializer function (`func init()`) | callable | zero; each source definition is independently owned and no `init` package binding exists |
| function literal | callable | zero |
| package `var` initializer (`var a, b = f()`) | initializer | zero or more non-blank declarations, in source name order; `var _ = f()` retains evaluation but declares nothing |
| bodyless function/method obligation | bodyless | exactly one |
| cataloged implicit definition | implicit | zero |
| package-synthetic cgo adapter | synthetic | exactly one and one callable signature |
| package-synthetic cgo type/data | synthetic | exactly one and no callable signature |

The declaration-name occurrence of `func init()` resolves to the owning
definition's name component, not to a fabricated declaration. Its body,
signature, source identity, and package-initialization reachability remain
independently conserved.

Evidence depth and provider selection are not semantic forms. In particular,
`external` and `intrinsic` cannot appear as `DefinitionSemantics` forms; the
Stage-1 `DefinitionSelection` and the semantic authority witness already own
those independent facts. Declaration records own declared value/type facts, so
`DefinitionSemantics` does not duplicate a second unordered declared-type set.

### Stage-2 Artifact And Authority Boundary

Stage 2 has two packages with non-overlapping ownership:

- `internal/language/semantic` owns the immutable target-independent schema,
  canonical semantic identities, validating constructors, package projection,
  and provider-semantic artifact format. It imports no `go/ast`, `go/types`,
  source loader, compiler orchestrator, or target package.
- `internal/language/frontend` is the sole transient checker consumer. It maps
  Stage-1 definitions and occurrences into the semantic schema before source
  finalization. It owns no second semantic schema and persists no toolchain
  object.

The Stage-1 structural provider artifact and Stage-2 semantic provider
artifact are separate authorities. Combining them would make the structural
layer own declaration types, bindings, variants, and operations; hydrating
provider bodies during ordinary compilation would defeat bounded acquisition.
The semantic artifact therefore binds the exact structural-artifact digest,
provider-contract fingerprint, selected toolchain/configuration, and package
input digests. It is package-sharded. Its resident manifest contains package
membership, exact definition and owned-declaration identity censuses, record
counts, and shard admission digests, but no declaration payload, type,
binding, or operation detail.

Ordinary compilation keeps local semantic records resident and projects at
most one certified semantic package at a time. Admission validates the whole
package shard before exposing any record. A semantic shard contains exactly
the package-owned `DefinitionSemantics`, `OccurrenceResolution`, declarations,
bindings, operations, and unsupported records, plus the complete canonical
type closure and per-type authority witnesses needed to validate that package
in isolation. Canonically equal type values may therefore occur in more than
one package shard; declarations and definitions may not. The artifact reports
this type-closure duplication explicitly. A shard cannot carry Stage-1
topology or TypeScript decisions. When a shard is projected, typed admission
revalidates every relationship, so corrupt detail remains invalid even if its
shard and outer artifact digests were recomputed. Ordinary manifest admission
does not claim to detect replacement of both artifact content and the
independently selected trusted digest; that is replacement of the authority,
not corruption under an admitted authority.

The ordinary Stage-2 verifier exact-joins the semantic model's package,
provenance, selected-authority, definition, and declaration censuses to the
source plan and admitted manifest without decoding provider-only semantic
shards. It decodes exactly the certified shards that also have selected local
semantic overlays, validates each merged package, then releases it. Later
phases may request additional package detail through the same bounded
projection API. The model reports certified-package count, mixed-overlay
count, shard loads, and maximum simultaneous residency separately; a
provider-only ordinary compilation has zero semantic shard loads.

When the structural-source plan selects local syntax, the semantic authority
is the one transient checker graph and the record carries
`CheckerAuthority(toolchain, package-input, structure, selection-fact
digests)`. When it selects a certified provider graph, the authority is
`CertifiedProviderAuthority(semantic-artifact digest, package-shard digest,
bound structural-artifact digest)`. If a package contains both local and
certified definitions, independently derived checker and provider records for
the overlapping definitions must be equal after authority is removed; the
source plan still selects exactly one authority per definition.

The toolchain `builtin` pseudo-package is the sole semantic owner of
predeclared declaration payloads. Ordinary package shards may reference their
pinned catalog identities and canonical types, but never duplicate predeclared
declarations. The language pseudo-package is materialized once from the
catalog exact-joined to `types.Universe`; it is not inferred from identifier
spelling.

### Canonical Semantic Identities

Semantic identities are constructor-only and machine independent:

- package-scope declarations use canonical package identity, closed object
  class, declared name, and generic owner;
- lexical bindings and labels use the exact Stage-1 occurrence that introduces
  their Go scope, their defining `OccurrenceID` when spelled, a closed binding
  role, and an ordinal. The scope occurrence is a file, definition root,
  signature, block, clause, or other cataloged scope owner; it is never a
  `go/types.Scope` pointer or reconstructed source path;
- unnamed receiver, parameter, result, and implicit bindings use that same
  scope occurrence, closed role, and ordinal rather than spelling or
  `token.Pos`;
- field and method identities include the canonical declaring package as well
  as owner type and name, so unexported members from different packages never
  conflate; field order is identity evidence, while method order is not;
- predeclared objects use the pinned predeclared-catalog identity;
- a spelled operation is identified by its owning definition plus its one
  executable `OccurrenceID` (exactly one operation may own that occurrence);
  an unspelled operation is identified by its implicit definition, closed
  implicit entry class, and ordinal, with no fabricated source occurrence; and
- types use a complete canonical descriptor with a full digest and
  collision check.

A canonical declaration payload does not own one privileged source occurrence.
Source declaration sites are the `OccurrenceResolution` records that resolve
to it. This distinction is required because equal unnamed structural types may
be spelled more than once:

```go
func F(left struct{ Value int }, right struct{ Value int }) {}
```

Both `Value` spellings introduce source evidence for the same canonical
unnamed-struct field (`struct-type ID + field ordinal`); neither spelling may
be discarded, and the declaration payload may not choose one as its identity
or authority. Package declarations, local declarations, and named members
normally have one declaring occurrence, predeclared declarations have none,
and structurally shared unnamed members may have more than one. Exact
occurrence-to-declaration joins own those cardinalities. Provider projection
selects declaration authority from declaring occurrences and owning
definitions, never from an arbitrary source field copied into the declaration
record.

A type descriptor preserves basic kind; alias versus defined nominal owner;
generic owner, binder role, ordinal, and arguments; array length; channel direction; signature
receiver, receiver type parameters, function type parameters, parameters,
results, and variadic state; struct field order, names, embeds, tags, and
package visibility; interface method names, declaring packages, signatures,
embeds, comparability, and normalized terms; and every recursively referenced
type identity. An interface's normalized structural type set has an explicit
closed state—`universe`, `finite(terms)`, or `empty`—because an empty term slice
cannot distinguish the universe from the empty set. Method descriptors contain
semantic method components rather than a member identity that points back to
the type being hashed, so anonymous interfaces have no circular identity
construction. Their callable signature excludes the receiver and receiver type
parameters because the containing method descriptor already owns that
relationship; declared method definitions retain receiver binding and receiver
type separately. Named, alias, and type-parameter references terminate structural
recursion at their nominal identity. No identity uses a pointer address, acquisition path,
`Type.String()`, source spelling without semantic owner, truncated digest, or
fallback poison value.

Type-parameter identity is owned by the generic language declaration, not by a
checker object or a lexical-binding record. Its closed owner is either the
canonical `SemanticDeclarationID` of an ordinary generic type/function/method
or the canonical `DefinitionID` of a source definition that intentionally has
no declaration (for example a legal blank-named generic function). The owner is
combined with a closed binder role (`type`, `callable`, or `receiver`) and the
zero-based source ordinal. Thus the `T` in:

```go
package dep

type Pointer[T any] struct{ value *T }

func (pointer *Pointer[T]) Load() *T { return pointer.value }
```

has stable identities derivable from `dep.Pointer` and `dep.Pointer.Load`
without retaining `dep` syntax in an importing package. The lexical
`SemanticBindingID` for a locally spelled `T` remains a separate binding fact
whose type is that canonical type-parameter `SemanticTypeID`; it is never the
type's identity owner.

This separation is required for export-data and instantiated-checker forms.
For example:

```go
package app

import "dep"

var current dep.Pointer[int]
```

may expose cloned `go/types.TypeParam` objects with no parent scope while the
`app` shard builds its complete canonical type closure. The frontend joins
such transient objects to the origin declaration, binder role, and ordinal.
Object addresses, `token.Pos`, parent-scope presence, and the spelling `T` are
corroborating lookup evidence only and never enter semantic identity. A
foreign generic type never requires a fabricated local binding or foreign
source hydration.

`go/types.Builtin` is not an ordinary callable declaration: both predeclared
functions (for example `append`) and package `unsafe` intrinsics (for example
`unsafe.Offsetof`) have call-site-specific semantics and no single valid
checker signature. Their declaration record therefore carries a closed
catalog-backed builtin identity and no `SemanticTypeID`. Every ordinary
declaration still requires exactly one canonical type. A builtin call operation
carries the exact call-site result, operands, expected type, and builtin
declaration reference. The complete real package `unsafe` surface
(`Pointer` plus its builtins) is exact-joined bidirectionally against one pinned
append-only member catalog carrying each member's type-or-builtin class.
Documentary-only `ArbitraryType`/`IntegerType` declarations are excluded;
neither source spelling nor their documentary signatures may be treated as
semantic types.
The non-root header occurrences inside those documentary declarations resolve
through the closed `IntrinsicContract` structural disposition. They preserve
the Stage-1 definition/header topology but carry no fabricated checker type;
the cataloged builtin declaration and each call site's operation evidence are
the semantic contract.

Go checker maps that intentionally attach both a definition and a use to one
identifier are represented by their language-defined primary resolution, while
the companion fact remains owned by its declaration/type record. In
`struct { T }`, `struct { *T }`, or an embedded qualified/generic type, the
defining identifier resolves to the embedded type and the field declaration is
owned separately. In `func (p *Pair[A, B]) M()`, each receiver
index resolves to its newly declared receiver type-parameter binding and the
receiver signature references that parameter type. Any other simultaneous
definition/use pair fails closed.

Instantiated checker objects are references, not new declarations. A generic
method object canonicalizes through `types.Func.Origin`; an instantiated struct
field canonicalizes through the original named owner and exact field ordinal.
All instantiations therefore reference one method/field declaration identity.
Pointer inequality between checker objects is never treated as semantic
duplication, while unequal origins or ordinals remain hard collisions.
Receiver type-parameter checker aliases exact-join through their canonical
defining `OccurrenceID`; an object pointer or scope parent is never an identity
component.

### Context And Operation Records

The transient occurrence index exact-joins every locally retained
`OccurrenceID` to the one source/checker node only for the Stage-2 build
window. Stage 1 populates that index during its existing structural and
executable traversals; Stage 2 may not reconstruct it with another AST walk.
The index is actively severed with the checker graph and never enters a
finalized artifact.

Frontend resolution is parent-directed. A parent assigns each child its
expected type, result arity, composite-literal owner, callable signature,
storage role, and lexical/control environment. The child consumes that context
plus its recorded Stage-1 grammatical role and checker evidence. A child never
walks to or scans its parent, and a later lowering never recomputes the
decision.

Stage 2 uses separate fixed linear passes because context assignment, checker-
object indexing, binding/capture extraction, and semantic resolution are
different relations with different owners. Each named pass visits a retained
occurrence at most once; it may use only constant-time or amortized indexed
scope/containment probes. The production work ledger counts input admission,
child-edge assignment, context assignment, object evidence, implicit bindings,
intrinsic evidence, captures, resolution, containment construction/probes,
member-type visits, scope probes, and record construction separately. Canonical
sort inputs and containment storage are separate counters. Combining these
relations into one stateful mega-visitor, hiding a scan inside one counted
visit, or reporting output records as construction work is forbidden.

For example, in:

```go
func Outer(value int) func() int {
	return func() int { return value }
}
```

the context pass assigns the literal and identifier roles, the object pass
joins the identifier to the parameter object, the capture pass records the
literal definition as a capturer, and the resolution pass materializes the
operations. Each pass sees each retained occurrence once; capture containment
is an indexed probe rather than a walk from the literal back to `Outer`.

One semantic operation record has exactly one closed origin:
`SourceOperation(OccurrenceID, kind, role, variant, token, span)` or
`ImplicitOperation(ImplicitDefinitionOp, ordinal)`. Source-origin fields are
forbidden on an implicit origin, and an implicit operation cannot acquire a
zero-length or synthetic source span merely to satisfy a source-shaped schema.
The operation record contains, as applicable:

- closed operation class, catalog kind, semantic variant, grammatical role,
  lexical token, source span, and owning definition;
- result type, expected type, exact constant, result arity, addressability,
  assignability, and value-versus-storage-place class;
- canonical declaration/binding/call/selection/generic-instance target;
- ordered operand occurrence identities and evaluation order;
- receiver adjustment, promotion, copy, conversion, interface, boxing,
  initialization, panic-boundary, and other cataloged implicit operations;
  and
- exact control, branch, range, switch, select, defer, panic, and recovery
  relationships where applicable. A branch's structural target is an
  `OperationID` in the same definition; its optional spelled label is a
  separate `SemanticBindingID`. One field never stands for both relations.

These fields state Go meaning only. An operation cannot contain a helper name,
TypeScript node, representation choice, output path, runtime dictionary,
emitter flag, or implementation owner.

Implicit effects nested under a source operation preserve multiplicity. Each is
keyed by closed implicit-operation kind, exact source/operand occurrence when
spelled, and ordinal, plus its source/target semantic types where applicable.
For example, `f(a, b)` may require two distinct `ValueCopy` effects; one
kind-only set entry is lossy and forbidden. An unspelled package-init
coordination operation is a top-level implicit-origin operation, not one of
these nested effects.

### Stage-2 Conservation

The following are blocking exact joins, performed package by package so
provider scale cannot force whole-artifact residency:

1. every Stage-1 `DefinitionID` ↔ one `DefinitionSemantics`;
2. every retained owner/header/boundary/executable occurrence ↔ one legal
   `OccurrenceResolution`;
3. every resolution reference and structural coverage target ↔ one
   declaration, binding, type, operation, or unsupported record in the same
   logical model, including predeclared declarations in the one language
   pseudo-package;
4. every full-semantic executable region ↔ complete operation/explicit-
   unsupported coverage, while every non-full definition ↔ zero executable
   operations; every typed implicit executable entry ↔ one non-source
   operation without a fabricated occurrence;
5. every checker object/type/selection/instance consumed by the producer ↔
   its canonical immutable record and authority witness; and
6. every local/certified overlap ↔ equal semantic content with exactly one
   selected authority.

The catalog owns the legal resolution classes for every construct kind, role,
and semantic variant. There is no default semantic operation, generic
structural fallback, or "unresolved but present" record. Unsupported is a
positive typed record with exact identity, evidence, and reason; it is not an
omitted operation.

## Whole-Program Facts

Analyses run over the complete selected model and produce one sealed fact set:

- exact direct and finite dynamic call graph;
- effects, escape, addressability, storage lifetime, and alias regions;
- value-copy and zero-value requirements;
- generic operation demand and concrete instantiations;
- interface implementer sets, conversion sites, and dynamic-type observations;
- object-family embedding, promotion, override, collision, and construction
  facts;
- map-key, equality, comparability, hashing, and ordering needs;
- nil, panic, recovery, initialization, concurrency, and external effects;
- source/API/test/extension roots and all dependency edge classes; and
- output cost attribution candidates.

Each fact kind has one owner and one query API. Parallel maps carrying pieces
of one logical record are forbidden. Population occurs before sealing; any
post-seal mutation is an invariant failure. Unknown facts select an explicit
unsupported disposition unless a declared conservative representation is
itself exact.

Semantic reachability is sealed before planning. It starts from explicit
executable, public API, selected test, reflection, and extension semantic roots
and traverses typed call, initialization, function-value, generic
instantiation, interface/dynamic, registration, and external-contract edges.
Every reachable semantic definition/operation has a root witness; every
unreachable one has an exact exclusion explanation. A conservative edge is
allowed only when its typed observation class makes that edge exact. Planning
and lowering receive only this reachable semantic set—“emit the selected
closure and prune later” is forbidden.

Post-completion implementation reachability remains separate. It traverses the
actual generated/manual/runtime/standard-library/external/extension artifacts,
validates that every semantic reachability edge was materialized, discovers
implementation-only helper/adapter/initialization edges, reports orphan manual
work, and determines publication/pruning. Neither graph substitutes for the
other.

## Immutable Program Plan

Planning chooses the simplest exact representation after all facts are sealed.
`ProgramPlan` is atomic: every semantically reachable declaration and operation receives a
complete validated plan or an error. It includes:

- type/storage/zero/copy/equality/key plans;
- function, method, receiver, closure, call, and generic plans;
- interface conversion, dispatch, assertion, equality, and RTTI plans;
- object-model inheritance/composition and construction plans;
- control, panic, defer, concurrency, and initialization plans;
- module/file/import/definition ownership;
- runtime, standard-library, external, manual, and extension obligations; and
- expected output-cost and exception attribution.

Related choices are one record. For example an external method obligation
stores identity, emitted slot, signature, receiver representation, and adapter
plan atomically. Consumers never recompute one field.

Planning is monotonic until fixed point and then frozen. Lowering cannot choose
between candidates, introduce a helper because printing is difficult, or retry
with a stronger representation.

## Typed TypeScript Construction

GoToTS owns a typed TypeScript AST sufficient for every generated construct.
Lowering traverses the immutable semantic model and requires one validated plan
record for every semantic operation it encounters. The plan references
semantic IDs instead of copying the semantic tree. Lowering may not import
`go/ast`, `go/types`, workspace loaders, analyzers, project profiles, or
whole-program fact producers, and it may not infer a missing plan.

One formatter owns TypeScript spelling and layout. Generator code outside that
formatter must not concatenate TypeScript source or hide templates in Go string
literals. Hand-maintained files under `runtime/`, `gostdlib/`, external
implementations, and extensions are ordinary TypeScript source and are not
generator templates.

Each implementation lowers into an isolated typed fragment containing its AST,
imports, definitions, initialization edges, helper requirements, and source-map
entries. The fragment merges into a module only after complete validation.
Outcomes are the closed variants `Emitted`, `KnownUnsupported`, and
`CompilerDefect`. Only `KnownUnsupported` may request a placeholder;
`CompilerDefect` fails the run. Failed fragments cannot leak imports, aliases,
helpers, counters, obligations, or partial source into another body.

Generated output uses strict ESM with explicit `.js` imports. Source files are
mirrored by default:

```text
example.com/project/ast/node.go  -> output/src/example.com/project/ast/node.ts
example.com/project/ast/types.go -> output/src/example.com/project/ast/types.ts
```

Files may merge only when a five-edge-class module graph proves that class
heritage, runtime initialization, value imports, type imports, and function
references cannot preserve the mirrored split. The merge and its reason are
recorded. Package-wide monoliths are not the default.

## Dependency Walls

The intended package dependency direction is:

```text
identity + language/catalog + language/typesemantics + scope/contract
        + language/semantic schemas + source/load
        <- scope/sourceplan
        <- language/structure
        <- language/selectionfacts
        <- scope
        <- language/executable
        <- language/frontend
        <- whole-program analysis
        <- plan
semantic + plan -> typescript/lower -> typescript/ast -> typescript/format

compiler/orchestrator coordinates the phases
completion and reachability consume finalized output/evidence
verification may inspect every layer; production layers never import it
```

Architecture tests enforce:

- no reverse imports across this graph;
- no corpus/integration imports from production packages;
- no `go/ast` imports outside `internal/source`,
  `internal/language/structure`, `internal/language/selectionfacts`,
  `internal/language/executable`, `internal/language/frontend`, and independent
  stage verification;
- no `go/types` imports outside `internal/source`,
  `internal/language/selectionfacts`, `internal/language/frontend`, independent
  stage verification, and the exact `internal/language/typesemantics` service;
- no source or project-profile imports below planning;
- no TypeScript string construction outside the formatter;
- no production import of verification packages; and
- no cycles hidden through a generic utility package.

## Repository Structure

The cleanroom implementation uses semantic ownership, not historical
directories:

```text
cmd/gotots/                  CLI wiring only
internal/compiler/           phase orchestration
internal/source/             workspace, toolchain, selected inputs, transient
                             syntax/checker lifetime, final severing
internal/scope/contract/     closed versioned provider/rule/fact-request
                             schema and validation; no source/graph state
internal/scope/sourceplan/   request-bound local/certified structural-source
                             planning; no definition-graph dependency
internal/scope/              per-definition provider/depth selection
internal/identity/           canonical source/definition/type/operation/
                             implementation IDs
internal/language/catalog/   closed Go construct catalog
internal/language/structure/ immutable Stage-1 schema and depth-independent
                             owner/containment/definition/site/header/boundary
                             graph producer; package-sharded provider storage,
                             identity census, and bounded package projection
internal/language/selectionfacts/ closed request-bound preselection semantic
                             facts, produced once and reused by Stage 2
internal/language/typesemantics/ exact shared Go type-set/core-type operations
internal/language/semantic/  target-independent semantic records
internal/language/executable/ parent-directed full-executable structural
                             traversal: occurrences, grammatical roles, and
                             executable regions; no go/types interpretation
internal/language/frontend/  sole checker-to-semantic materializer: definition
                             semantics, occurrence resolutions, variants,
                             bindings, types, and implicit operations
internal/stagecheck/         blocking in-pipeline independent stage joins,
                             run synchronously between phases; distinct from
                             internal/verify offline/certification gates
internal/analysis/           sealed whole-program facts
internal/plan/               immutable representation/implementation plan
internal/typescript/ast/     typed target nodes
internal/typescript/lower/   plan-to-AST conversion
internal/typescript/format/  deterministic source printer
internal/typescript/service/ parser, resolver, and strict checker bridge
internal/completion/         ownership and structural reconciliation
internal/reachability/       full graph, traversal, explanation, pruning
internal/external/           typed external contract planning
internal/extension/          finalized-evidence extension assembly
internal/evidence/           ledgers, fingerprints, reports
internal/verify/             independent gates
runtime/                     minimal language runtime TypeScript
gostdlib/                    generated declarations plus manual TS bodies
integration/                 project-independent acceptance corpora
testdata/                    construct and lifecycle fixtures
calibration/                 reviewed Go and hand-written TS pairs
profiles/                    data-only project requests
schemas/                     machine artifact schemas
```

Directories are dependency/ownership boundaries. Files are construct families,
not one file per AST node and not numeric shards. Examples include
`expression_index.go`, `statement_assignment.go`, `generic_demand.go`, and
`interface_plan.go`.

Maintained implementation files must be semantically focused and below 600
physical lines. Packages or files named `util`, `utils`, `helper`, `helpers`,
`misc`, `obsolete`, `compat`, `fallback`, or `v2` are prohibited. Shared code is
named for the semantic concept it owns.

## Compiler Orchestration

`internal/compiler` sequences phases and handles cancellation and artifacts. It
contains no semantic cases. `cmd/gotots` parses flags and invokes the compiler;
there is one binary and one compilation route.

Heavy stages use bounded concurrency, disk-backed staging, memory preflight,
and resumable breadcrumbs. A retry may resume the same immutable request; it
may not change semantics. Two heavy whole-product jobs must never be stacked.

Publication writes immutable versioned roots and atomically replaces one
current pointer only after all required gates pass. A failed generation leaves
the prior product untouched.

User errors, unsupported constructs, and compiler defects are distinct typed
errors carrying phase, project, package, source/implementation/occurrence IDs,
construct kind, role, and span. Error strings are rendered only at the CLI;
they are never parsed to select behavior. Invalid internal state fails at its
constructor or phase boundary rather than being downgraded to unsupported.

## Project-Independence Gate

Production code and generated runtime templates must contain no source-project
package path, repository name, type name, or method-name condition except names
defined by the Go language, selected target ABI, or explicit external contract.

The gate performs both broad scans and behavioral mutations:

- reject `typescript-go`, Microsoft corpus paths, and acceptance-fixture names
  under production packages;
- compile a relocated fixture whose module identity is changed;
- compile at least two unrelated projects through the same pipeline;
- swap corpus order and prove plans depend only on semantic evidence;
- inject an unrecognized construct and require the catalog boundary to fail;
  and
- mutate a project profile name without changing output semantics.

The acceptance corpus may reveal a missing generic abstraction. It can never
authorize a corpus-specialized branch.
