# GoToTS Specification

## Authority

These five files are the complete governing specification:

1. `README.md` — product boundary and vocabulary;
2. `architecture.md` — direct compiler architecture and project structure;
3. `translation.md` — contextual construct translation contract;
4. `verification.md` — proof and regression requirements; and
5. `delivery.md` — implementation order and completion criteria.

Historical branches, plans, reports, and implementations are evidence only.
They cannot override this specification. Requirements use **must** and
**must not**.

## Mission

Given a valid selected Go project, toolchain, build configuration, explicit
compilation profile, and environment contracts, GoToTS produces deterministic,
readable, strict ESM Tsonic-flavored TypeScript. The canonical artifact retains
Go intent through ordinary TypeScript and accepted target-neutral marker
contracts. A selected target separately lowers that checked artifact to its
executable representation. Any intentional departure from Go semantics is
named in the target profile and may not be described as exact.

It is a general Go compiler. `typescript-go` is an acceptance corpus, not a
production dependency, language authority, package-name exception, or source
of privileged translation rules.

GoToTS is independently buildable and owns every Go semantic decision. Apart
from the pinned TS-Go toolchain used for TypeScript AST construction and
printing, it never invokes another compiler to analyze Go. Its canonical output
may import accepted public marker declarations from `@tsonic/core`; GoToTS does
not copy those declarations or import a target plugin/runtime. TSTS and the
selected target are downstream consumers, not Go semantic truth owners.

## One Compilation Model

```text
selected files and package graph
        |
        v
standard Go parser + one coherent go/types graph
        |
        v
one parent-directed, context-aware emission walk
        |
        v
generated typed Go bindings for the pinned TS-Go AST protocol
        |
        v
pinned TS-Go printNode API
(real TS-Go decoder, node factory, and printer)
        |
        v
canonical strict Tsonic-flavored TypeScript modules
        |
        v
TSTS finalized marker facts plus exact authored occurrences
        |
        v
selected target-owned AST lowering and executable artifact
```

The source program is represented only by the selected Go AST and its
`go/types` evidence. The target program is represented only by strongly typed
values generated from TS-Go's official external AST protocol. The pinned TS-Go
process decodes those values into its real AST nodes and prints them with its
real printer. There is no compiler-defined source inventory artifact, semantic
IR, operation graph, whole-program plan, handwritten target tree, local
formatter, or target-text fallback inside GoToTS.

Target lowering is a separate compilation boundary, not a hidden GoToTS IR.
TSTS checks the exact immutable canonical text, retains its TS-Go-contract AST,
and finalizes typed marker facts on exact AST nodes. A target transforms that
same AST and submits the result to the stable printer boundary. It must not
reparse, join source ranges, reread files, recognize marker spellings, re-enter
the checker, or infer Go meaning. Bootstrap printing uses pinned TS-Go; a later
TSTS-native printer may replace it without changing target semantics.

This does not prohibit ordinary compiler coordination. Deterministic names,
scope builders, imports, target declaration assemblies, diagnostics, references
into the Go graph, root requests, and observable-contract dependency edges are
allowed. They must not restate the source program as a second model or become a
second semantic truth owner.

Mutable target builders, declaration assembly, and placement are owned by the
root emitter. Handlers receive immutable scope identities/capabilities and
return typed root requests; they do not mutate arbitrary ancestors. One
declaration owner may reconstruct its own typed TS-Go nodes while the
compilation is open. References record exactly which observable facet of
another target artifact they consumed. The root reconstructs reverse dependents
only when structural comparison says that subscribed facet changed. The final
target file is sealed and printed only after all requests and affected
artifacts reach a fixed point.

## Source-Shape Conservation

Every target declaration that represents a Go function, method, function
literal, defined callable, interface method, or environment callable preserves
the selected `go/types.Signature` as its public/source-facing contract.

- Go value parameters remain in the same order and cardinality.
- A value receiver may become TypeScript `this`; a pointer receiver may become
  one explicit first receiver parameter so nil can enter the method body.
- A Go variadic parameter remains one semantic parameter. Its exact target
  representation may be a represented Go slice or a TypeScript rest parameter,
  but it may not create an additional source argument.
- Every result remains the direct synchronous projection of the selected Go
  result tuple. A source callable never gains `Promise`, `async`, `await`, a
  scheduler, recovery authority, bridge, policy, or capability parameter.
- Source type-parameter arity is preserved. Logical/storage distinctions do
  not become extra public type parameters.

Compiler mechanics may appear only in compiler-owned private support artifacts
that do not claim a Go source identity. Such artifacts may be selected by exact
Go identity and imported statically, but they may not leak through package
assembly, environment declarations, function values, interfaces, or ordinary
source calls. In particular, translated source contracts never contain hidden
operation functions, recovery authorities, provider policies, bridge sets,
representation facets, profile selectors, or digest-named public variants.

When direct TypeScript cannot implement a representation-dependent generic
operation, the compiler concretizes only the reached declaration/type operation
at its exact `go/types.Instance`. A concretization has the same source value
parameters after type substitution and is an internal generated definition;
it is not a second generic ABI. If a required open case has neither a direct
static form nor a finite exact concretization, compilation fails explicitly.

## Vocabulary

- **Go construct:** a grammar form represented by Go syntax, such as an
  assignment, call, declaration, index, receive, return, or function literal.
- **construct occurrence:** one concrete Go AST node in a dispatchable semantic
  role. Child syntax consumed as part of its parent's indivisible rule is not
  forced through an independent handler.
- **context:** facts supplied by the parent handler, such as expression versus
  statement position, assignment target, expected arity/type, lexical scope,
  and evaluation boundary.
- **construct case:** a Go AST form together with its parent role, relevant
  type evidence, expected results, and evaluation context. `values[key]` in a
  one-result assignment and in a comma-ok assignment are different cases.
- **semantic evidence:** the selected toolchain's `go/types` facts for the
  existing Go AST, including object bindings, types, constants, instances,
  selections, and signatures.
- **dispatcher:** a category-level router that selects exactly one semantic
  owner for one requested node. It never recursively walks the node.
- **handler:** the one contextual translation owner for a construct family.
- **child contract:** the handler-owned list of meaningful direct children,
  their closed roles, dispatch categories, order, and semantic boundaries.
- **child emitter:** the narrow callback interface through which a handler asks
  the root emitter to dispatch one child with an explicit context.
- **emission result:** typed TS-Go protocol AST values plus explicit root
  requests and diagnostics. It is target output under construction, not an IR.
- **root request:** a closed typed request for placement, a use-dependent
  declaration obligation, or an artifact dependency. Combined requests form an
  immutable persistent DAG; only the root consumes it, and shared subgraphs are
  never flattened into per-consumer copies.
- **placement request:** a typed request to insert an import, declaration,
  helper, temporary, or statement at a legal preferred target scope.
- **declaration requirement:** a closed typed request, keyed by the
  authoritative `go/types` declaration identity, asking that declaration's one
  target owner to add or revise a use-dependent target obligation before the
  containing file is sealed.
- **target artifact:** the complete root-owned, pre-seal TS-Go AST assembly for
  one authoritative Go definition. Its semantic owner is the only code allowed
  to reconstruct it.
- **observable facet:** one closed consumer-visible part of a target artifact's
  canonical TS-Go AST contract, such as callable signature, constructor
  surface, instance type surface, static surface, or exported value surface.
  Function bodies and explicitly typed member/value initializers are not
  observable facets; inference-dependent target declarations are rejected. A
  body-only artifact may consume dependencies while providing zero facets.
- **artifact dependency:** a typed edge from one target artifact to one
  observable facet of another, recorded when the consumer constructs a target
  reference. Containment and source reachability are not artifact dependencies.
- **artifact reconstruction:** one transactional replacement of an artifact's
  complete TS-Go AST roots, root requests, dependency set, and observable
  contract. Exact unchanged facets do not notify consumers.
- **declaration assembly:** the root-owned, compilation-local target state for
  one source definition: already-created typed TS-Go protocol nodes plus its
  deduplicated declaration requirements, requests, dependencies, and canonical
  observable contract. It is not a source model or IR.
- **generated support module:** a GoToTS-owned TypeScript module containing
  deduplicated Go-language aliases or behaviorally real Go runtime operations
  required by generated files. It is constructed through TS-Go AST like every
  other output file. Target-neutral semantics already owned by `@tsonic/core`
  use that canonical marker declaration instead of a copied support API.
- **source-facing contract:** the canonical GoToTS callable or type surface
  corresponding to one selected Go declaration. Its value and type-parameter
  shape obeys Source-Shape Conservation even when generated modules import a
  private concretization or provider facade to implement it. A closed
  executable target may project its representation only by rewriting every
  affected producer and consumer while preserving Go parameter cardinality,
  order, and observable behavior.
- **concretization:** a private generated definition reconstructed directly
  from one Go generic declaration at one exact reached `go/types.Instance`
  because its body needs representation-dependent operations that TypeScript
  cannot express over the open type parameter. It carries no operation
  dictionary and is keyed by Go identity plus exact type arguments.
- **canonical representation rule:** the direct rule choosing the ordinary
  TypeScript shape or accepted target-neutral marker that exactly records one
  Go type, method, interface, value, or operation before target lowering.
- **target lowering rule:** a target-owned, fact-driven transformation from a
  canonical marker occurrence to that target's executable representation. It
  may project a complete closed executable contract, but it cannot mutate the
  canonical artifact, add/drop/reorder Go value parameters, change observable
  Go behavior, or rediscover semantics by spelling.
- **compilation profile:** the immutable compilation-wide selection of Go
  semantic tradeoff axes retained in canonical source. Target representation
  and optimization choices belong to a separate target profile. Canonical
  files in one compilation cannot mix selections.
- **target profile:** the selected downstream target and its immutable
  representation/optimization policy. It consumes finalized facts and may not
  alter canonical source artifacts or Go behavior outside a named equivalence
  envelope. Any executable type projection requires complete-flow evidence and
  atomic producer/consumer rewriting.
- **Go build profile:** the loader-owned selected Go identity, `GOOS`,
  `GOARCH`, `CGO_ENABLED`, and sorted build tags. The identity binds the
  selected `GOVERSION`, executable bytes, and content-addressed complete
  `GOROOT` snapshot without persisting machine paths. Commands consume only
  that sealed snapshot. The selected version must equal the `go/ast` and
  `go/types` frontend version compiled into GoToTS; defaults still come from
  the selected Go executable rather than ambient shell state. Cross-target selection changes source
  files, sizes, standard-library contracts, and runtime constants as one
  atomic choice.
- **manual obligation:** an exact generated declaration whose implementation
  must be supplied manually.
- **standard-library provider:** the GoToTS-owned `@gotots/gostdlib` package.
  Its public ESM subpaths mirror Go standard-library import paths, its named
  exports preserve Go declaration names, and its implementation satisfies the
  exact selected-`GOROOT` contract. The selected host backend is implementation
  metadata and never appears in the public package name.
- **true external:** unavailable or host/native behavior represented by an
  explicit contract rather than inferred from import spelling.
- **source implementation:** a project-selected, certified strict-ESM module
  that owns the final TypeScript implementation of one exact source package
  contract instead of the translated package bodies. It is selected by
  canonical package identity, never package spelling in compiler code.
- **equivalence envelope:** the explicit observable-behavior boundary for one
  source implementation. `exact` requires Go-equivalent results. A narrower
  project-specific envelope may change an internal algorithm only after its
  consumers prove the changed value is not externally observed; public
  signatures, determinism, required equality behavior, and failure behavior
  remain certified.

## Product Boundaries

All imported packages obey the same Go language. Toolchain metadata separately
identifies:

- workspace and source-available dependency packages, which are translated
  unless an exact certified package implementation is selected;
- standard-library declarations, whose selected-`GOROOT` contracts are
  generated and whose behavior is completed in reusable `gostdlib`;
- toolchain pseudo-packages and intrinsics, which have explicit compiler
  ownership; and
- true external, native, platform, `cgo`, or unsupported boundaries, which
  receive exact contracts and explicit placeholders.

No import-path prefix decides these classes.

A source-available package may contain a selected bodyless native declaration.
That declaration remains source-owned and emits one exact typed throwing
placeholder plus one canonical obligation until an explicit provider
implementation replaces it. It is not converted into an ambient package and
does not prevent the rest of the source graph from being generated and
typechecked. Executing the placeholder fails by its Go declaration identity;
publication requires every reachable placeholder to be gone.

A project may instead select one certified source implementation for an exact
source package contract. The final output then contains the implementation
module and no translated package body, placeholder, or second owner for that
package. This is a general project mechanism, not a compiler conditional for a
known package. The selected implementation preserves the generated package
assembly surface and is parsed into the pinned TS-Go AST before it becomes a
target file.

An equivalence envelope may relax an algorithm only when the selected product
proves the relaxed result is internal. For example, an internal content-cache
hash may use a different fast deterministic 128-bit hash when no hash value is
serialized, exposed, or compared with a specified digest. The implementation
must still make equal inputs equal, keep the advertised collision width, and
preserve cache correctness. A checksum, protocol digest, persisted key, test
vector, or public return value is observable and therefore requires exact
behavior.

## Non-Negotiable Results

- Every encountered construct is handled or fails with a typed unsupported
  diagnostic carrying source identity and context.
- Production dispatch never descends automatically. The selected handler
  accounts for every direct field and delegates each semantically independent
  child exactly once under its closed child contract.
- Context-sensitive meanings are decided by the parent handler plus
  `go/types`, never by text or guesses.
- Every emitted TypeScript construct exists first as an exact typed value of
  TS-Go's official AST protocol and is printed by pinned TS-Go itself.
- Definitions are owned once and output growth remains proportional to source
  complexity, not interface implementer count or package count.
- Ordinary declaration and module names are deterministic, source-derived, and
  readable. Digests remain artifact/evidence identities; they do not appear in
  ordinary generated source names or paths.
- Generated code uses no `any`/`unknown` recovery, dynamic semantic lookup,
  `.call`, `.apply`, `.bind`, dynamic import, prototype patch, or source-text
  patching.
- Go-specific aliases, support declarations, and runtime operations are defined
  within the generated product. Accepted target-neutral semantics import their
  canonical `@tsonic/core` declarations. GoToTS never imports a target plugin or
  target runtime.
- A marker is valid only when exact Go evidence selects its canonical provider
  declaration, TSTS emits the corresponding finalized fact, and the selected
  target consumes that fact. A local same-spelled function or no-op runtime body
  cannot substitute for semantics.
- Correct behavior within the declared profile, strict static typing,
  maintainable source shape, generated size, typecheck cost, generation cost,
  and runtime cost are simultaneous acceptance dimensions.
