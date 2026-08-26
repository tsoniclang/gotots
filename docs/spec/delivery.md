# Delivery

## Development Model

Delivery is construct-driven, test-first, and dependency-ordered. Milestones
are capability checkpoints, not alternate compiler phases. Every milestone
uses the same direct Go AST/`go/types` to typed TS-Go AST architecture for
canonical source, followed by the one selected fact-driven target boundary.

For each construct family:

1. add the smallest valid Go fixture and expected artifact shape;
2. observe the focused test fail at the owning unsupported boundary;
3. implement the highest shared semantic owner;
4. delete any superseded path in the same change;
5. prove canonical AST, finalized-fact totality, and exact-node ownership;
6. run focused target-AST, strict, differential, and mutation proof;
7. inspect canonical and executable artifacts and cost;
8. run broader gates before checkpointing.

No milestone may introduce a temporary semantic IR, hidden source ABI,
compatibility shim, corpus-specific override, or implementation that a later
milestone is expected to discard.

Callable representation work lands as one vertical capability: the closed
projection contract, automatic and authored-signature evidence, the canonical
signature facet, every definition/reference consumer, superseded-path deletion,
artifact inspection, differential and mutation proof, and measured product
delta. A direct-call-only rewrite or a package-specific manual-call adaptation
is not a deliverable.

## 0. Native Target And Loader

Install:

- one configuration/CLI owner for optional Go and TS-Go executables plus the
  explicit project cache root;
- sealed, path-free semantic identities for the exact selected Go,
  content-addressed complete `GOROOT` snapshot, and pinned TS-Go build;
- exact selected-tool routing through package loading, provider/source
  certification, strict compilation, and AST printing;
- pinned TS-Go schema and generated typed protocol bindings;
- real `printNode` encode/decode/print round trip;
- explicit Go build profile and selected package graph;
- exact loader-owned `//go:embed` declaration, pattern, file, and payload joins;
- deterministic module/name/import/scope builders;
- root placement and revisable artifact fixed point;
- architecture walls.

Exit: a minimal valid package prints canonical strict ESM only through the one
selected pinned TS-Go; TSTS finalizes every selected marker fact on its exact
AST node; the selected target directly transforms the TS-Go-contract AST and
produces strict executable output. Tool/root drift, a foreign TS-Go build,
non-overlay package metadata, repeated per-command root hashing, ambient
tool/scratch inheritance, schema
mutation, ambient build-profile drift, second parser, range join, text emitter,
spelling-selected marker, source reread, or duplicate artifact owner fails.

## 1. Core Direct Emission

Install test-first:

- files/packages/imports/comments;
- constants, variables, basic types, and function declarations/literals;
- identifiers, literals, selectors, calls, conversions, built-ins;
- unary/binary expressions;
- assignments, returns, blocks, if, for, switch;
- tuples/multiple results and variadics.

Exit: every admitted form has one contextual handler, closed child coverage,
strict printed artifacts, focused Go-versus-TS execution, and no unknown form.

## 2. Value Families

Install:

- exact zero/copy/conversion/equality owners;
- defined types and aliases;
- structs/classes and demand-created members;
- arrays, slices, strings, pointers, and maps;
- composite literals, indexing, slicing, range, and addressability;
- all integer profiles and both evaluation-order profiles.

Exit: focused family matrices and mutations pass; direct representations stay
direct; carriers/helpers appear only on exact demand; size grows with source
operations rather than all possible operations.

## 3. Methods, Embedding, And Interfaces

Install:

- value receiver as instance `this`;
- pointer receiver as class-owned unbound/static first parameter;
- method values/expressions;
- exact promotion and conditional native inheritance;
- canonical interface metadata and one reached adapter;
- O(1) interface dispatch, assertions, type switches, equality, and map keys;
- canonical runtime type descriptors and typed reflection value views;
- `TypeFor`, `TypeOf`, `ValueOf`, and reached `reflect.Type`/`Value` operation
  families without host reflection or erased payloads.

Exit: nil/copy/promotion/dynamic-value differentials pass, ordinary concrete
calls remain statically selected, and implementer-count scaling leaves call
sites constant. Reflection descriptors exact-join the selected `go/types`
facts, dynamic and static type observations share one identity, and descriptor
size is linear in reached type structure.

## 4. Generics And Iterator Functions

Install:

- generic functions, aliases, named types, and methods;
- explicit/inferred instances from `types.Info.Instances`;
- direct generic emission where TypeScript is exact;
- private exact concretization where an open representation-dependent
  operation is not expressible;
- all admitted range-over-function signatures.

Exit: source value/type arity is exact; no operation parameters/objects exist;
direct body size is independent of call count; concretizations exact-join
necessary reached instances; recursive and unsupported-open cases terminate
deterministically.

## 5. Function Control

Install:

- Go panic carrier and runtime faults;
- immediate defer capture, fixed slots for single-entry direct sites, dynamic
  LIFO stacks only for repeated/conditional/goto-controlled sites, named
  results, and panic replacement;
- ordinary and private deferred entries for direct `recover`;
- exact-signature typed registry for transported deferred callables;
- labels, fallthrough, break/continue, and non-structural goto assembly.

Exit: source callable signatures contain no recovery parameter; every direct
and dynamic defer form passes the recovery-directness matrix; functions without
control demand remain artifact-stable.

## 6. Serial Execution

Install the one fixed synchronous execution contract:

- one direct callable ABI for functions, methods, literals, callable values,
  interface methods, callbacks, deferred entries, and package initialization;
- immediate serial execution of `go` calls;
- buffered channel send, receive, close, range, and atomic ready/default
  `select` operations;
- typed serial-blocking panics for every operation that would suspend;
- the narrow close-observer capability required by synchronous context and
  signal providers; and
- synchronous provider/facade/kernel certification.

Exit: generated and provider-facing callable surfaces contain no `Promise`,
`async`, `await`, awaitable union, scheduler, blocked-operation queue, or
execution-profile variant; ready channel differentials and every blocking,
close, nil, and panic boundary pass.

## 7. Language Closure

Derive and exact-join the selected toolchain's full active AST/token/builtin
domain to production ownership. Close remaining valid:

- declaration/expression/statement forms;
- blank identifiers by parent role;
- package aliases, dot/blank imports, and command packages;
- labels and control composition;
- toolchain-version feature differences.

Exit: zero unknown valid selected constructs, no duplicate owner, and every
unsupported form has a typed boundary.

## 8. Environment And Standard Library

Install:

- deterministic generated package/runtime/environment filesystem;
- selected-`GOROOT` contracts;
- exact typed placeholders and obligation ledgers;
- settled environment use-demand and implementation-route evidence on the root
  obligation projection;
- provider implementation dispositions with certified private
  dependency/capability closure;
- the exact used-provider closure gate;
- standalone strict-ESM `@gotots/gostdlib`;
- standalone strict-ESM `@gotots/externals` modules for selected true-native
  boundaries;
- provider certificate and generated static facades;
- one provider-owned, certified scalar ABI with selected native width;
- true external/native/cgo boundaries;
- reachable-obligation deletion gate.

`@gotots/externals` owns a small, hand-maintainable public TypeScript surface,
an exact module/export seed, and a generated canonical certificate. Its public
subpaths mirror Go import paths (for example
`@gotots/externals/golang.org/x/sys/unix.js`). The provider may depend on the
exact certified `@gotots/gostdlib` package and GoToTS runtime, but generated Go
callables never gain provider-policy parameters. Local product assembly links
the package normally through ESM package resolution; publication is a separate
later action.

Public provider APIs and generated facades preserve source arity. Generic,
interface, and recovery support remains statically linked in private
facade/kernel artifacts. No runtime policy object, capability argument,
per-method effect variant, or ambient fallback survives. A semantic protocol
may retain at most one uniform synchronous certificate for a Go identity;
ordinary direct bindings are not duplicated.

Capability views used only inside the provider and views permitted in generated
bridges are separately certified. Only the latter enter reverse interface
demand. Delivery fails if a provider-internal view appears in generated output,
if an exported view lacks a usage, if a direct bridge selects a canonical view,
if a canonical bridge selects a direct view, or if either usage selects an
incompatible base or target provider profile.

Exit: provider package independently strict-typechecks and executes; exact
contract/facade joins pass; a linked representative product uses one runtime
identity and has zero reachable placeholders, zero used provider placeholders,
and zero unresolved used profile boundaries under the exact settled closure. A
satisfied generated obligation or certified binding is not implementation
proof; publication scope is the selected closure, not the entire standard
library.

Delivery generates and strict-typechecks the complete product runtime under
all integer profiles. The provider build is independent of that product
selection: provider source contains no import of `@gotots/runtime/scalars.js`,
and its certificate pins the provider representation plus native width.
Linked-product proof executes both equal-carrier and converting facades and
rejects stale or mismatched provider scalar contracts before target sealing.
Runtime parity is certified only under a profile that can preserve every
reached integer-dependent identity and control decision. If the product reaches
exact fixed-width 64-bit arithmetic, the runtime replay selects at least
`fixed64-bigint`; if it reaches exact native 64-bit overflow, it selects
`bigint`. A successful `number` typecheck is not runtime-equivalence evidence.

## 9. Product Proof

For each acceptance corpus, including TS-Go:

1. select the exact build and compilation profiles;
2. generate all source-available packages;
3. generate and certify every provider/external obligation;
4. print all canonical TypeScript through pinned TS-Go;
5. strict-check canonical source and exact-join every selected marker to one
   finalized TSTS fact and authored occurrence;
6. lower through the selected target's AST and strict-typecheck the complete
   executable product under bounded resources;
7. link it locally with `gostdlib`, the generated Go runtime, and any demanded
   target runtime;
8. execute representative CLI/library entry points;
9. compare behavior to the selected Go build;
10. report canonical/executable size, largest expansions, time/RSS, runtime, and
   unresolved obligations;
11. preserve exact failure artifacts.

Compile-only is not runtime completion. Runtime completion requires an actual
generated entry point to execute with expected observable output and error
behavior.

## Source-ABI Conservation Delivery

Every source-facing callable is mechanically joined to its selected
`go/types.Signature`. The accepted implementation has one ordinary source
entry and only demand-created private implementation artifacts:

- directly expressible generic bodies remain direct;
- representation-dependent generic bodies may use reached exact private
  concretizations, never public operation/capability parameters;
- recover support may use one private deferred entry while the ordinary source
  entry remains unchanged;
- indirect callables recursively use the same direct synchronous signature as
  their source callable type; and
- provider boundaries use exact static facades whose public calls retain the
  source arguments.

Every new callable family must pass the same source-shape gate before it is
admitted. Operation dictionaries, recovery parameters, provider policies,
hidden representation type parameters, public callable-profile variants,
runtime Promise inspection, call-site-first profile discovery, and source-ABI
fallbacks are forbidden. A replacement is atomic by owner: it installs the
new owner, migrates every producer and consumer, and deletes the superseded
path in one checkpoint.

## Performance Selection And Source Implementations

Performance delivery follows measured ownership rather than corpus rules:

1. the versioned project config, `-c`/`--config`, and CLI overrides resolve to
   one immutable semantic compilation contract;
2. canonical GoToTS output chooses the smallest exact target-neutral shape,
   requests semantic machinery only on demand, and uses readable
   source-derived declaration/module names;
3. a certified package-atomic source implementation may replace a package only
   after independent contract/equivalence proof, with every translated body
   absent from the installed artifact;
4. executable-representation optimizations belong to the selected target and
   consume finalized exact-node facts plus complete-flow evidence; and
5. the next change is selected from measured source-size, typecheck, memory,
   startup, or runtime evidence at the highest owner that eliminates the class.

Each change starts with a failing owner-level contract and ends with focused
shape, strict-typecheck, differential/equivalence, mutation, final-artifact,
human-source, and cost evidence. Full generation and runtime jobs are batched
at milestone boundaries under the repository memory guard. Faster runtime
alone cannot accept a package implementation or representation change.

## Checkpoint Evidence

Every pushed checkpoint records:

- exact parent and clean feature-branch revision;
- governing spec revision and TS-Go/toolchain/provider fingerprints;
- changed semantic owner and deleted path;
- focused/full/race test denominators;
- generated artifact examples;
- strict typecheck and runtime evidence;
- absolute and parent-delta size/time/RSS;
- mutations caught;
- broad deletion searches;
- honest remaining obligations.

No remote branch is force-pushed or deleted. Publication is a separate,
explicitly approved action.
