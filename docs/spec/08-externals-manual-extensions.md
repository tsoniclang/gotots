# Externals, Manual Bodies And Extensions

## Three Separate Ownership Classes

The following are distinct and cannot substitute for one another:

1. **external contract** — a library declaration outside selected owned Go
   source;
2. **accepted manual body** — one selected Go implementation whose declaration
   and semantics remain generator-owned; and
3. **product extension** — behavior with no upstream Go body, assembled through
   a generated typed seam.

Every record names exactly one class.

## External Packages

Every imported package outside selected owned roots is external only when it is
not in a declared outside-universe root. An edge into LSP, fourslash, editor,
or another outside root is a scope error and never an external obligation. The
Go standard library and other permitted imported packages are external.
GoToTS translates references to their declarations but does not implement
library behavior.

External identity derives from canonical Go package/object evidence:

- module or standard-toolchain origin;
- package path and version/replacement identity;
- object kind and declaration identity;
- receiver, signature, fields, methods, generic constraints, and constants;
- transitive type dependencies; and
- source edges requiring the object.

Filesystem and emitted module names use canonical digest-based encoding, not
unchecked import text.

## External Declaration Closure

For every referenced external object, GoToTS computes complete declaration-level
type closure. It includes named types, aliases, underlying structures,
constraints, methods, embedded fields, parameter/result types, constants, and
instantiated arguments needed to type selected code.

The closure excludes external function bodies. Missing declaration evidence
blocks rather than widening to an untyped value.

## External Effects

A Go signature does not reveal mutation, retention, address escape, panic,
blocking, callback, concurrency, or alias-return effects. Those effects come
from a versioned reviewed contract.

Absent effects are conservative:

- pointer and container arguments may escape;
- mutable arguments may be retained and changed where the type permits;
- callbacks may be retained and invoked;
- calls may panic or block when the contract does not exclude it; and
- returned values may alias permitted inputs.

Conservative effects can force a stronger representation. They never invent
library behavior.

Conservatism does not authorize an unbounded product-wide universal carrier.
If an unknown effect cannot be isolated by an exact static boundary adapter and
would otherwise erase useful representation distinctions across an unbounded
closure, the external edge is unimplemented until its reviewed effect contract
is refined.

Effects include semantic observations, not only mutation: an external that
reports a dynamic type, compares identity, observes nil versus empty, reads
string bytes, or retains a slice header declares that observation explicitly.

## External Representation ABI

After fixed-point planning, every external edge receives one static
representation ABI record covering:

- concrete parameter/result storage and canonical nil encoding;
- copy, alias, ownership, retention, and mutation behavior;
- slice/map header and backing behavior;
- text versus arbitrary-byte string encoding;
- pointer/storage-location identity and access;
- interface dynamic-type tokens and permitted payload branches;
- panic/host-failure conversion, blocking, callback, and initialization; and
- any explicit generated boundary adapter and its semantic/cost proof.

The emulation binding either implements that exact ABI or fails assembly.
Binding by source signature alone is insufficient. A binding may declare a
small accepted ABI family, but the assembled edge still statically selects one
member; no runtime representation probing or owner registry is permitted.

## Typed External Stubs

Each canonical external package/version receives one deterministic generated
contract module. Object exports are allocated from canonical object IDs; the
selected emulation module implements those same static exports. Opaque external
values use generated branded types and can be inspected only through declared
external operations.

The translator-capability artifact emits static declarations and deterministic
stub implementations for unresolved external functions. A stub:

- has the exact generated TypeScript signature;
- records canonical external identity and contract hash;
- throws `GOTOTS_EXTERNAL_UNIMPLEMENTED` immediately if invoked;
- contains no fake return value or host-library substitution; and
- is reported as unresolved.

The selected product artifact replaces each reachable stub through a static
binding manifest. Product publication rejects a reachable unresolved stub.

Variables and addressable external storage use typed get/set/location contracts
only when required by source operations. Constants are emitted from exact Go
constant evidence.

## External Emulation Binding

A binding identifies:

- external contract identity and hash;
- implementation module and export;
- exact static signature hash;
- effect-contract hash;
- implementation revision/integrity;
- initialization requirements; and
- semantic oracle evidence.

Assembly imports the selected implementation directly. There is no runtime
registry, name lookup, package fallback, or speculative host adaptation.

## Manual Body Purpose

Manual ownership is an honest completion mechanism for a body whose typed IR is
complete but automatic lowering is not yet suitable. It is appropriate for
rare difficult implementations when a general runtime would be
disproportionate.

Manual ownership never hides an unknown declaration, signature, operation, or
effect.

## Manual Ownership Unit

The only manual unit is a complete function/method body or explicit synthetic
package-initializer body. Declarations, signatures, types, imports, module
ownership, source maps, and seams remain generated.

Partial statements, expressions, line ranges, regex replacements, textual
markers, and signature edits are not ownership units.

## Manual Body Record

Each accepted body records:

- canonical Go implementation ID;
- source signature hash;
- source semantic-body hash;
- semantic IR hash;
- lowering-plan hash;
- baseline generated-body hash or blocking-plan hash;
- accepted canonical TypeScript body-AST hash;
- dependency and effect hashes;
- canonical declaration/helper references and import requests;
- semantic-oracle and representation-verifier IDs;
- owner and reason;
- source and output profile hashes; and
- acceptance revision.

The authoritative body artifact is a versioned canonical serialization of one
typed TypeScript body AST. It is stored outside generated roots and keyed by
canonical Go identity.

References inside that AST bind to canonical Go declarations, generated helper
IDs, and explicit module requests. Allocated local/import spelling is resolved
during assembly and is never stored as semantic identity.

## Manual Promotion Workflow

The supported workflow is:

1. generate the declaration and baseline body or blocking diagnostic;
2. allow a developer to edit the complete body in the assembled module for
   review;
3. parse and typecheck that module structurally;
4. compare the observed canonical body hash with baseline and accepted hashes;
5. explicitly promote the complete body;
6. extract it into the authoritative body artifact with reviewed effects; and
7. regenerate the module from Go inputs plus the accepted artifact.

Promotion requires a Go differential oracle for the complete body, structural
verification against the representation plan, and class-level necessity for
any custom mechanism used by the body. Manual ownership cannot bypass copy,
nil, panic, boundary, or staticness rules.

There is no automatic promotion. An edited generated body without a matching
accepted record is drift and blocks. Regeneration never relies on retained
generated output.

## Manual Drift

A manual body becomes stale when its source signature, source semantic body,
IR, lowering plan, dependencies, effects, profile, extension seams, or accepted
artifact changes. Staleness blocks the affected closure and reports every
changed hash.

Formatting changes do not alter canonical AST hashes. Intentional semantic
edits require explicit promotion.

## Product Extensions

Product extensions add TSTS-specific observation, fact, provider, session,
virtual-module, and embedding behavior that has no upstream Go body. Extension
implementation stays in separately owned modules outside generated roots.

GoToTS generates only the typed core contracts and static bridge calls needed
by accepted extension seam data.

No-extension generated output remains exact to pinned Go. Extension-enabled
behavior is governed by a versioned product-extension contract that states each
intentional addition or deviation and supplies its own semantic oracle. An
extension cannot silently weaken the no-extension contract.

## Seam Specification

A seam is declarative input keyed by canonical Go declaration or semantic
operation identity. It records:

- seam category and exact typed payload/result;
- semantic placement in the IR evaluation phases: before operand evaluation,
  after operand capture, before checks, before mutation commit, after success,
  loop/branch edge, or lifecycle boundary;
- cardinality and deterministic ordering;
- enabled build/profile conditions;
- permitted compiler-state access and mutation;
- error, cancellation, and fact-lifecycle behavior;
- extension module/export identity; and
- source and proof hashes.

Placement never uses an emitted local name, raw source fragment, line number,
or regex.

An around seam has a typed continuation with declared zero-or-one invocation
cardinality, result/panic/control-flow effects, and deterministic nesting. A
complete operation replacement is allowed only when the product-extension
contract owns that exact operation, declares whether original operands/checks/
mutation execute, and differentially proves the intentional behavior. There
is no unrestricted middleware-style replacement.

A seam cannot change a synchronous generated ABI to async, add undeclared
allocation, or access compiler state outside its typed capability contract.

## Mid-Body Seams

Entry/exit hooks are insufficient for every compiler extension. A mid-body seam
attaches to a semantic IR operation or control-flow edge. Lowering composes a
typed bridge at that operation before TypeScript emission.

If an upstream update removes, duplicates, or changes the operation, seam
cardinality/type validation blocks regeneration. The extension is never moved
to a nearby textual location.

For an `accepted-manual` body, required seams are canonical typed seam nodes in
the manual AST. Missing, duplicated, reordered, or stale seam nodes block the
body; assembly never textually injects a seam into manual source.

## Static Extension Assembly

An enabled extension imports and calls a concrete typed bridge. A disabled
profile emits no extension call or pays only an accepted static absence cost.
Generated core never discovers extensions at runtime.

Extension modules typecheck against generated contracts. Missing modules,
signature drift, duplicate exclusive seams, ambiguous ordering, and undeclared
state mutation block assembly.

## Compiler State Mutation

Extension seams may read or mutate compiler state only through typed contracts
whose lifecycle and invariants are declared. Mutable fields corresponding to
shared compiler state remain direct fields on the generated structures when
that is the selected model; extensions cannot introduce shadow side tables to
replace them.

## Ownership Gates

Independent verification proves:

- every external reference resolves to one contract and selected binding state;
- every manual body resolves to one current canonical artifact;
- every extension seam resolves to one typed assembly decision;
- no unit has multiple ownership classes;
- generated source contains no text patches or runtime owner selection; and
- unresolved external, stale manual, missing extension, and unimplemented core
  closures are reported separately.
