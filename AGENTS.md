# Agent Notes (GoToTS)

The canonical workspace policy in `../AGENTS.md` applies. This file contains
only GoToTS-specific ownership, translation, structure, and verification rules.

## Governing Architecture

`docs/spec/` is authoritative. GoToTS is a general Go-to-TypeScript compiler;
no acceptance corpus receives privileged behavior.

GoToTS is an independently buildable general compiler. It owns every Go
semantic decision and never invokes Tsonic or a target compiler to rediscover
Go meaning. Its canonical output contract is strict ESM Tsonic-flavored
TypeScript: ordinary TypeScript plus public target-neutral marker types and
calls selected from `@tsonic/core` by exact declaration identity. Those marker
contracts are shared Tsonic authority, not copied GoToTS declarations.

Executable target code is a separate consumer artifact. TSTS checks the exact
immutable canonical source, retains its TS-Go-contract AST, and finalizes
marker facts on exact AST nodes. A selected target transforms that same AST
without spelling lookup, source-range joining, reparsing, or checker re-entry.
GoToTS does not import a target plugin or target runtime, and the target does
not reinterpret Go source.

The compilation architecture is deliberately direct:

```text
selected Go packages
    -> Go AST plus one coherent go/types graph
    -> one context-aware emission walk
    -> typed generated bindings for the official TS-Go AST protocol
    -> pinned TS-Go printNode using its real decoder/factory/printer
    -> canonical strict ESM Tsonic-flavored TypeScript

canonical TS-Go AST plus finalized TSTS facts on exact nodes
    -> selected target-owned TS-Go AST transformation
    -> pinned TS-Go bootstrap printer, later replaceable by TSTS printing
    -> executable target artifact
```

There is no custom source inventory artifact, semantic IR, operation graph,
program plan, lowering IR, or text-emission fallback. A handler may inspect the
selected Go AST, query the existing `go/types` evidence, and construct TS-Go AST
nodes. It must not copy those facts into a second semantic model.

Allowed compiler state is limited to references into the Go AST/type graph,
deterministic target-name ownership, target lexical-scope builders, explicit
root requests, canonical observable TS-Go contract facets, facet-specific
reverse dependencies, diagnostics, and already-created typed TS-Go protocol
AST values. Such state coordinates emission; it must not become an alternate
representation of the source program.

## Translation Invariants

- Every encountered Go construct has exactly one contextual handler or one
  explicit unsupported disposition. Unknown forms fail before formatting.
- The selected toolchain's parser, AST, and type checker own Go validity.
  Production code does not copy the Go grammar or generate a second parser.
- Production dispatch never recursively walks a node. The selected semantic
  handler accounts for every direct child under a closed role, order, context,
  and boundary; it consumes inseparable syntax itself and delegates only
  semantically independent children, exactly once.
- Parent handlers supply child roles and expected result shape. Children never
  recover context by source-text scanning, spelling inference, or an alternate
  parent graph.
- Semantic questions use the selected `go/types.Info`, `types.Package`, and
  method/selection evidence directly. Do not rerun the checker.
- Translation results contain typed TS-Go protocol AST values plus typed root
  requests; they are not a target-independent intermediate
  representation.
- Evaluation-dependent statements remain at the exact execution boundary.
  Imports and preferred-static helpers go to file scope. Placement is selected
  by one deterministic policy, not by whichever caller happens to emit first.
- Dynamic imports are forbidden. Generated code must not use
  `Function.prototype.call`, `apply`, `bind`, prototype patching, reflection,
  or dynamic shape inspection.
- TypeScript is generated only through exact typed bindings and an encoder
  mechanically generated from TS-Go's pinned official external AST protocol.
  The pinned `tsgo --api` `printNode` endpoint performs real AST decoding and
  printing. No local formatter, source fragments, templates, token-stream
  emitter, handwritten target AST, inferred wire shape, forked `internal`
  import, or post-format text patching.
- Go embedding is not assumed to be TypeScript inheritance. `extends`,
  `implements`, native methods, receiver functions, and interface dispatch are
  selected only when their Go call, nil, copy, promotion, and method-selection
  behavior remains exact.
- Ordinary statically selected receiver methods must not accidentally become
  target-language virtual dispatch. Genuine interface dispatch must be O(1)
  per call and must not expand with implementer count.
- Every source-facing callable preserves its selected Go value-parameter
  order and cardinality. A value receiver may become `this`, a pointer receiver
  may become one explicit first parameter, and a variadic slot may use its one
  exact target representation. Recovery authorities, operation dictionaries,
  provider policies, bridges, scheduler/effect state, and other compiler
  mechanics must not appear as translated source parameters. Source generic
  arity is likewise exact; representation facets and callable profiles stay in
  private support artifacts or finite exact concretizations.
- Definitions are emitted once. References may repeat. All helper and import
  requests are deduplicated by typed ownership, never rendered text.
- Every generated binding in a source-bearing lexical or member scope is
  allocated by the canonical name owner against all visible authored,
  imported, and generated bindings. Counter spelling is not collision proof.
  Literal target-only names are confined to structurally closed synthetic
  scopes whose complete binding set is owned and verified locally.
- Every revisable target artifact is keyed by authoritative Go identity and
  reconstructed only by its semantic owner. References record closed,
  facet-specific dependencies on the provider's canonical observable TS-Go AST
  contract. The root requeues reverse dependents only when exact structural
  comparison changes a subscribed facet; implementation-only changes do not
  propagate. Reconstruction replaces the complete pre-seal artifact and its
  dependency/request set transactionally. Text patching, mutable-node sharing,
  spelling keys, unconditional rescans, and non-convergent cycles are forbidden.
- Canonical output remains strict ESM and may import only accepted public
  target-neutral marker declarations from `@tsonic/core`; it never imports a
  target plugin or target runtime. Go-specific support declarations and
  runtime behavior remain GoToTS-owned unless a shared marker contract exactly
  owns that semantic class.
- A marker is admissible only when exact `go/types` evidence selects its
  canonical provider declaration, TSTS finalizes the corresponding typed fact,
  and every selected target either lowers that fact or rejects it explicitly.
  Marker spelling, a local same-named declaration, or a no-op JavaScript body
  never establishes semantics.

## Environment Ownership

All Go imports share the same language semantics. Toolchain metadata—not import
spelling—distinguishes workspace source, source-available dependencies,
standard-library packages, toolchain packages, and true external boundaries.

- Source-available packages are translated normally unless one exact package
  contract has a project-selected certified source implementation; that
  implementation becomes the sole final target owner.
- Every load resolves and records one explicit Go build profile; ambient shell
  `GOOS`, `GOARCH`, `CGO_ENABLED`, `GOFLAGS`, and tags never select source.
- The selected `GOROOT` defines the standard-library declarations.
- `gostdlib` supplies manually completed behavior against generated contracts.
- True externals receive exact typed contracts and explicit placeholders.
- A selected bodyless source declaration emits one exact typed throwing body
  and one canonical obligation; it never becomes an executable `declare`.
- Generated primitive aliases and reusable runtime modules are GoToTS-owned
  output, not external environment contracts.
- Runtime helpers exist only for Go behavior that direct TypeScript cannot
  preserve exactly.
- Extensions receive explicit typed emission context and TS-Go factories; they
  cannot patch text or create a second analysis path.

## Code Discipline

- Parallel agents are forbidden unless the user explicitly authorizes them for
  the specific task. Task size, urgency, or separability does not imply
  permission.
- Begin implementation only after the relevant specification examples,
  ownership, failure behavior, and verification are explicit.
- Every foundation capability and construct case begins with its focused test
  observed failing at the owning missing/unsupported boundary.
- Semantic handlers live under the recursive
  `internal/emit/<domain>/<semantic-owner>/<sub-owner-as-discovered>` structure.
  Root emitter files contain orchestration only; handlers import the narrow API
  and target boundary, not the root emitter or sibling handlers.
- Reuse archived code only when it is independently the ideal direct-emitter
  design. Never restore a subsystem merely because it previously passed tests.
- Use validating constructors and closed enums where invalid states otherwise
  become representable. Errors are typed; error strings never select behavior.
- Do not create `v2`, `legacy`, `compat`, `fallback`, `util`, `utils`,
  `helper`, `helpers`, or `misc` packages/files.
- Keep maintained non-generated implementation files focused and under 600
  physical lines.
- Keep generated artifacts reproducible from checked-in schema and inputs.

## Verification

Every substantial construct family closes before dependent work begins:

1. source examples cover ordinary, contextual, and adversarial forms;
2. handler-totality and ownership checks fail on omissions and duplicates;
3. constructed TS-Go AST is schema-valid and reparses identically;
4. generated TypeScript passes strict typechecking before execution;
5. Go and TypeScript behavior is compared differentially;
6. production-path mutations fail at the owning gate;
7. actual generated artifacts, including the twenty largest expansions, are
   inspected;
8. source bytes, generated bytes/AST nodes, typecheck time/RSS, generation
   time, runtime, and helper/definition counts are reported with parent deltas;
9. broad searches prove forbidden and superseded paths are absent.

Heavy jobs run one at a time with bounded concurrency, timeout, disk-backed
output, and an OS memory ceiling. Preserve failure artifacts. Never retry an
OOM with the same unbounded command.

Passing tests alone is insufficient. Behavior must be exact within the
explicitly selected compilation profile; every intentional profile boundary
must be named and must never be reported as exact Go behavior. Strict static
TypeScript, source-shaped output, bounded generated size, bounded typecheck
cost, and bounded runtime are simultaneous correctness requirements.

## Repository History

- `archive/*` branches are historical evidence, never production dependencies.

## Outward Reviews

Review messages are engineering artifacts. Bind them to the exact inspected
revision and governing spec, distinguish facts from proposed work, preserve one
active endpoint, and perform a separate pre-send review. If a note to the user
is needed, put it before a `Message to Team` heading; otherwise print only the
complete copy/paste-ready team message.
