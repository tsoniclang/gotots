# Delivery

## Development Model

Delivery is construct-driven, test-first, and dependency-ordered. Milestones
are capability checkpoints, not alternate compiler phases. Every milestone
uses the same direct Go AST/`go/types` to typed TS-Go AST architecture.

For each construct family:

1. add the smallest valid Go fixture and expected artifact shape;
2. observe the focused test fail at the owning unsupported boundary;
3. implement the highest shared semantic owner;
4. delete any superseded path in the same change;
5. run focused AST, strict, differential, and mutation proof;
6. inspect generated artifacts and cost;
7. run broader gates before checkpointing.

No milestone may introduce a temporary semantic IR, hidden source ABI,
compatibility shim, corpus-specific override, or implementation that a later
milestone is expected to discard.

## 0. Native Target And Loader

Install:

- pinned TS-Go schema and generated typed protocol bindings;
- real `printNode` encode/decode/print round trip;
- explicit Go build profile and selected package graph;
- exact loader-owned `//go:embed` declaration, pattern, file, and payload joins;
- deterministic module/name/import/scope builders;
- root placement and revisable artifact fixed point;
- architecture walls.

Exit: a minimal valid package prints strict ESM only through pinned TS-Go; a
schema mutation, ambient build-profile drift, text emitter, or duplicate
artifact owner fails.

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
- immediate defer capture, LIFO unwind, named results, panic replacement;
- ordinary and private deferred entries for direct `recover`;
- exact-signature typed registry for transported deferred callables;
- labels, fallthrough, break/continue, and non-structural goto assembly.

Exit: source callable signatures contain no recovery parameter; every direct
and dynamic defer form passes the recovery-directness matrix; functions without
control demand remain artifact-stable.

## 6. Cooperative Concurrency

Install the explicit race-free `cooperative` profile:

- channel representation and typed live queues;
- goroutine scheduler and program settlement;
- send, receive, close, range, and atomic select;
- direct callable effect propagation;
- one canonical `Awaitable` ABI per indirect signature and interface method;
- package/program asynchronous initialization where demanded.

Exit: no per-method or combinatorial callable-profile variants, hidden effect
type parameters, result shape tests, or all-function async conversion; full
channel/scheduler differentials, deadlock/panic tests, and cost gates pass.

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
interface, cooperative, and recovery support remains statically linked in
private facade/kernel artifacts. No runtime policy object, capability
argument, per-method profile variant, or ambient fallback survives. Required
semantic protocols may retain at most one uniform direct certificate and one
uniform cooperative certificate for a Go identity; ordinary direct bindings
are not duplicated.

Capability views used only inside the provider and views permitted in generated
bridges are separately certified. Only the latter enter reverse interface
demand. Delivery fails if a provider-internal view appears in generated output,
if an exported view lacks a usage, if a direct bridge selects a canonical view,
if a canonical bridge selects a direct view, or if either usage selects an
incompatible base or target callable profile.

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
4. print all TypeScript through pinned TS-Go;
5. strict-typecheck the whole generated product under bounded resources;
6. link it locally with `gostdlib` and the generated runtime;
7. execute representative CLI/library entry points;
8. compare behavior to the selected Go build;
9. report source/generated size, largest expansions, time/RSS, runtime, and
   unresolved obligations;
10. preserve exact failure artifacts.

Compile-only is not runtime completion. Runtime completion requires an actual
generated entry point to execute with expected observable output and error
behavior.

## Active Source-ABI Replacement

The current implementation contains superseded machinery that violates
Source-Shape Conservation. It is replaced in this dependency order:

### A. Freeze The Gate

- mechanically compare every source-facing declaration/call to
  `go/types.Signature`;
- add failing examples for generic operation parameters, recovery parameters,
  provider policy objects, hidden representation type parameters, and public
  callable-profile variants;
- add static artifact searches and mutation controls.

### B. Generic Directness

- keep directly expressible generic bodies;
- install exact reached-instance concretization for representation-dependent
  bodies;
- migrate generic functions/methods/values/provider calls;
- delete operation-parameter propagation, capability ordering, operation
  objects, and their public/provider ABI fields.

### C. Deferred Entry

- emit ordinary exact source entry plus private recovery entry only where
  demanded;
- install typed exact-signature registry for transported function values;
- migrate direct, method, interface, generic, provider, and asynchronous
  deferred invocation;
- delete optional recovery parameters/facets from every source callable and
  public provider signature.

### D. Canonical Indirect Awaitability

- use one `Awaitable` function/interface result mapping under cooperative
  mode;
- unconditionally await indirect calls;
- retain exact direct callable effects;
- migrate named callable types, interfaces, adapters, generic nested
  callables, package state, and environment contracts;
- delete source/profile variants, public profile exports, hidden `$Value`
  parameters, and runtime Promise inspection.

### E. Static Provider Facades

- derive the complete directional interface/callback obligation set from the
  selected Go surface and inspected provider project before emission;
- exact-join every obligation to the ordinary binding or one private facade,
  and reject every missing, duplicate, extra, mixed-effect, or result-triggered
  certificate;
- make generated calls retain source arguments;
- generate one typed facade per selected provider boundary;
- import exact bridges/guards/private kernels statically;
- migrate stateful types, callbacks, nested interfaces, generic instances, and
  deferred entries;
- delete `CanonicalBoundaryPolicy`, runtime policy objects, profile matrices,
  hand-authored overrides, ordinary-binding fallback, and call-site-first
  profile discovery.

### F. End-To-End Reproof

- regenerate provider certificates and all affected artifacts;
- pass source-shape, schema, architecture, strict, differential, mutation,
  race, and cost gates;
- broad-search every superseded mechanism;
- generate, whole-product strict-typecheck, link, and execute TS-Go;
- checkpoint and push only clean feature-branch evidence.

These outcomes are atomic by owner: each replacement installs the new owner,
migrates every producer/consumer, and deletes the old path in the same
checkpoint. A newly observed blocker changes this sequence only when concrete
evidence invalidates the architecture; its directive must pass WCBUBWHB and
the review gate first.

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
