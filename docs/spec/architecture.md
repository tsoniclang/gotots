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
| Workspace load | Go workspace, toolchain, build config | typed package universe | target decisions |
| Construct inventory | syntax and `go/types` evidence | exhaustive occurrences | lowering |
| Semantic analysis | occurrences plus grammatical roles | Go semantic model | TypeScript shape |
| Whole-program analysis | complete semantic model | sealed facts | emission |
| Planning | semantic model plus facts | immutable total `ProgramPlan` | source rediscovery |
| TS lowering | semantic model plus `ProgramPlan` | typed TypeScript AST | semantic decisions |
| Baseline formatting | typed TS AST | canonical generated files | manual preservation |
| Completion | new baseline plus prior editable AST | reconciled mixed AST | guessing ownership |
| Reachability | full current implementation graph | retained product graph | text search |
| Verification | sources, models, plans, artifacts | gate evidence | repairing output |
| Publication | verified staged root | atomic current product | partial replacement |

Each phase consumes a complete typed artifact. A later phase cannot call an
earlier analyzer to fill a missing fact. Errors propagate immediately and do
not create partial records that consumers might interpret as valid.

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

Package, declaration, and type analysis treats ordinary workspace, dependency,
standard-library, and toolchain source uniformly. Body selection remains an
explicit per-implementation disposition: standard-library behavior is manually
completed, source-available module behavior is ordinarily automatic, and
intrinsic/external cases remain exact obligations. Every excluded body has that
typed disposition; package provenance cannot silently omit occurrences or
replace per-implementation ownership.

The compiler-supported catalog version, selected toolchain version, each
module's `go` directive, and the effective language version of each source file
are separate facts. Every occurrence uses the effective file version reported
by typed toolchain evidence, including file-specific version effects. A
workspace-wide maximum, lexical version comparison, or catalog maximum cannot
admit a construct in a file governed by an older language version.

## Complete Go Construct Catalog

The selected Go version has one machine-readable catalog containing:

1. every grammar/syntax form represented by `go/ast` and `go/token`;
2. every context-dependent semantic variant exposed by `go/types.Info`;
3. all predeclared identifiers and built-ins; and
4. implicit operations such as zeroing, copying, receiver adjustment, method
   promotion, interface conversion, initialization, and panic boundaries.

The catalog is authoritative Go code, not parallel JSON maintained by hand.
Generated reports may render it. Tests reconcile it against the selected
toolchain's concrete AST node types, token set, built-in universe, and Go
language-version features.

Every catalog kind has:

- a stable enum value and descriptive name;
- applicable grammatical roles;
- required typed evidence;
- produced semantic operation;
- allowed support dispositions; and
- focused positive, negative, and mutation fixtures.

A terminal enum sentinel and exact-size tables make omitted kinds fail tests.
There is no default `other`, textual prefix classifier, or generic recursive
fallback.

## Context-Aware Construct Analysis

The semantic unit is:

```text
syntax node
  + parent-assigned grammatical role
  + expected type and arity
  + lexical/storage/control environment
  + exact go/types evidence
```

The analyzer uses controlled recursive descent. An exhaustive dispatcher sends
each concrete construct to one construct-family visitor. The parent visitor
assigns the role for every child edge. A child visitor must not inspect its
parent, source spelling, or emitted context to infer meaning.

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

## Immutable Program Plan

Planning chooses the simplest exact representation after all facts are sealed.
`ProgramPlan` is atomic: every selected declaration and operation receives a
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
identity + language/catalog
        <- language/semantic
        <- source/load + language/analyze
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
- no `go/ast` or `go/types` imports outside `internal/source` and
  `internal/language/analyze`;
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
internal/source/             workspace, toolchain, profile, inventory
internal/identity/           canonical source/type/operation/implementation IDs
internal/language/catalog/   closed Go construct catalog
internal/language/semantic/  target-independent semantic records
internal/language/analyze/   context-aware typed visitors
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
