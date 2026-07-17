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

The editable translation workspace emits static declarations and deterministic
stub implementations for unresolved external functions. A stub:

- has the exact generated TypeScript signature;
- records canonical external identity and contract hash;
- throws `GOTOTS_EXTERNAL_UNIMPLEMENTED` immediately if invoked;
- contains no fake return value or host-library substitution; and
- is reported as unresolved.

An implementation may be supplied by editing that exact typed body in place;
the post-format body-hash contract below then classifies it as manual. Product
assembly derives its direct static import/call graph from typed source and
selects that one implementation. There is no user-authored registration record,
parallel external implementation registry, or runtime selection path. Product
publication rejects a reachable unresolved stub.

Variables and addressable external storage use typed get/set/location contracts
only when required by source operations. Constants are emitted from exact Go
constant evidence.

## External Emulation Binding

A derived binding record identifies:

- external contract identity and hash;
- implementation module and export;
- exact static signature hash;
- effect-contract hash;
- implementation revision/integrity;
- initialization requirements; and
- semantic oracle evidence.

The normal implementation workflow edits the generated typed throwing stub in
place. That body may directly call a separately maintained typed emulation
module; its import and binding record are derived from the resolved TypeScript
AST. The developer writes no attachment manifest. Assembly imports the selected
implementation directly. There is no runtime registry, name lookup, package
fallback, or speculative host adaptation.

A typed wrapper around erased dispatch is still erased dispatch and is
forbidden. This is not an external binding:

```ts
export function Clone(value: string): string {
  return goExternalCall("strings.Clone", [value] as unknown[]) as string;
}
```

Nor may the implementation be recovered through
`Map<string, Function>`, `Record<string, Function>`, a string-keyed method
table, `Function`, `unknown[]`, or a cast from an untyped dispatcher. The
generated signature does not make the hidden call edge static.

An unresolved translator-capability stub is a direct typed function that
throws. A selected-product binding is a direct typed ESM import selected at
assembly:

```ts
import { Clone as cloneImpl } from "@gotots-extern/strings.js";

export function Clone(value: string): string {
  return cloneImpl(value);
}
```

Function values and interface calls follow the same rule: their finite typed
target representation and call ABI must be explicit. Open or erased target
sets are unimplemented rather than routed through a universal dispatcher.

## Manual Body Purpose

Manual ownership is an honest completion mechanism for a body whose typed IR is
complete but automatic lowering is not yet suitable. It is appropriate for
rare difficult implementations when a general runtime would be
disproportionate.

Manual ownership never hides an unknown declaration, signature, operation, or
effect.

## Manual Ownership Unit

The smallest independently preserved manual unit is one complete executable
body: function, method, constructor, getter, setter, function-valued
initializer, function literal, or explicit synthetic package initializer. A
manual edit to a statement or expression makes its containing body manual; it
does not create a textual patch unit.

Generated and manual bodies may coexist in the same file and in the same
class. This is required for low-ceremony completion of isolated difficult
bodies. One body still has exactly one implementation; a generated fallback
and a manual implementation never coexist at runtime.

Generated declarations, signatures, fields, source maps, extension seams, and
module ownership remain generator-owned when they correspond to selected Go.
Editing one of those is not a body override and fails structural reconciliation.
Imports are regenerated from the resolved references of both generated and
manual ASTs, so a developer does not maintain a second import manifest.

A file without the mandatory generated-file header is manual source. Its
declarations are not disposable generated output. When such a declaration is
intended to satisfy a selected Go object, reconciliation must match it
unambiguously to the newly generated canonical module/export and exact
signature; otherwise it is a separate product/manual declaration or an error.

## Generated Baselines And Body Hashes

Every generated file begins with the versioned generated-file header. Every
independently owned generated body has an adjacent marker carrying:

- canonical implementation ID;
- marker schema version; and
- lowercase SHA-256 of the exact body text after pinned deterministic
  formatting.

The body byte range begins at the opening `{` and ends after the matching `}`.
It includes whitespace and comments inside that range and excludes the marker,
signature, imports, and generated-file header. The formatter emits UTF-8 with
LF line endings before hashes are computed. The generator first prints and
formats the marker-free module, hashes each body range, inserts the markers,
and verifies that marker insertion did not change any body range.

For example, the generated baseline may contain:

```ts
// Code generated by GoToTS; generated bodies are identified by verified body hashes.
// gotots:body v=1 id="pkg/math.Add" sha256="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
export function Add(left: int32, right: int32): int32 {
  return left + right;
}
```

The immutable accepted baseline proves what marker and body hash the generator
actually produced. A marker in the editable workspace is never trusted by
itself. Changing the marker cannot disguise a body edit.

The baseline manifest also carries each complete generated-file content hash as
a fast unchanged-file check. That file hash is never the ownership authority:
when it differs, reconciliation parses a copy, applies the pinned formatter in
memory, and computes each current body hash from the normalized body range. A
formatting-only edit may therefore normalize back to generated ownership;
semantic or comment changes inside a body remain visible.

Ownership is derived without a user-authored manifest:

- verified generated header plus a current body hash equal to the immutable
  baseline marker means generated and disposable;
- verified generated header plus a differing body hash means manual override;
- verified generated header plus an absent marker means manual override;
- no verified generated header means the file and its declarations are manual;
- malformed, duplicate, wrong-identity, or baseline-inconsistent markers are
  invalid and block rather than guessing.

Every generated function-like body receives a marker, including nested
function literals represented independently by the census. Reconciliation is
structural and deepest-first. If an outer raw body hash changed only because a
marked nested body changed, the nested body is the manual unit and the outer
body remains generated. A change outside nested body ranges makes the outer
body manual. This avoids overlapping ownership while retaining the required
post-format raw hash for every body.

The marker always records the accepted generated baseline hash, never a hash of
the manual replacement. A manual edit therefore remains visibly divergent.
Canonical TypeScript AST hashes remain separate semantic evidence; they do not
replace the post-format ownership hash.

## Throwing Placeholders

When typed IR is complete but automatic lowering is unavailable, the editable
workspace contains the exact generated declaration and a typed throwing body:

```ts
// gotots:body v=1 id="pkg/checker.solve" sha256="<generated-placeholder-hash>"
export function solve(input: Input): Result {
  throw new Error("GOTOTS_MANUAL_UNIMPLEMENTED");
}
```

Unresolved external operations use
`GOTOTS_EXTERNAL_UNIMPLEMENTED` instead. The placeholder has the exact static
signature, is machine-counted, and cannot return a fabricated value. Replacing
only the throw with ordinary typed code changes the post-format hash and is the
complete act of creating a manual implementation. No JSON, annotation,
registration call, dependency list, ownership declaration, or separate
promotion command is required.

A reachable placeholder is never a runnable product implementation. It may
exist in an explicitly incomplete editable/analysis workspace; publication
withholds its dependency closure until the body is implemented and accepted.

## Automatic Manual Reconciliation

Regeneration is structural and transactional:

1. attest the immutable prior generated baseline and parse the current editable
   workspace;
2. classify body ownership from headers, baseline markers, and post-format body
   hashes;
3. extract changed manual AST units in memory with their resolved symbol,
   dependency, effect, and extension-seam evidence;
4. generate the new baseline from the new Go pin into an empty staging root;
5. discard every old generated body rather than copying or adapting it;
6. join each manual unit to the new canonical Go declaration and signature;
7. overlay only unambiguous, structurally valid, reachable manual bodies;
8. derive imports and dependency edges from the assembled typed AST;
9. run all required structural, staticness, semantic, test, and performance
   gates; and
10. publish atomically.

The old editable tree is never a semantic input to automatic translation. It is
read only to recover manual ownership relative to the attested old baseline.
No generated body, helper, import, or declaration survives because it happened
to exist in that tree.

## Automatic Dependency And Reachability Graph

The developer writes TypeScript, not graph metadata. GoToTS derives the graph
from typed Go IR, generated TypeScript AST, and manual TypeScript AST. It
includes:

- static imports, calls, constructors, fields, methods, and package init;
- function values, closures, callbacks, deferred calls, and stored callables;
- finite interface dispatch and generic instantiation targets;
- external call-in/effect contracts and product/export roots;
- product-extension seams and generated bridge calls; and
- selected test roots.

An edge is canonical object identity, not source spelling. A dynamic target is
accepted only when whole-program typed analysis proves a finite complete set.
An unresolved target is conservatively reachable or blocking; it is never
discarded through a name or text heuristic.

Manual code may call generated code, other manual code, or typed external
contracts normally. Those relationships are discovered from resolved symbols.
There is no parallel manual dependency file.

## Manual Status And Regeneration Delta

Each regenerated manual unit receives exactly one derived status:

- `reachable-current` — joins the new declaration/signature and all evidence is
  current;
- `reachable-placeholder` — selected but still has a generated throwing body;
- `reachable-stale` — joins structurally, but source body, semantic IR,
  lowering plan, effects, extension seams, or proof inputs changed;
- `reachable-missing` — a required implementation has neither automatic nor
  manual behavior;
- `unreachable` — no current selected root reaches it;
- `orphaned` — its prior canonical Go declaration is absent from the current
  pin or has no unambiguous new join;
- `automatically-lowerable` — a manual body still overrides a body the current
  translator can now generate; or
- `invalid` — malformed marker, signature/declaration drift, unresolved static
  reference, overlapping ownership, or failed validation.

Reports list every status by canonical identity and show old/new source,
signature, IR, plan, generated-baseline, manual-AST, dependency, effect, seam,
and evidence hashes as applicable. These are generator-derived ledgers, not
user-maintained inputs.

`reachable-stale`, `reachable-missing`, `reachable-placeholder`, and `invalid`
block affected product closure. `unreachable` and `orphaned` manual source is
preserved by default, excluded from product assembly, and reported; it does not
silently become generated or disappear.

## Reset, Acceptance And Pruning

When automatic lowering later supports a manually implemented body, the manual
body remains the sole selected implementation. Regeneration reports
`automatically-lowerable` but does not add a generated alternative. An explicit
identity-targeted reset command may replace it with the newly generated body;
the command shows a dry-run diff and requires the current manual hash to match
the reviewed input. Reset changes ownership, not runtime selection.

Manual acceptance requires the same strict typecheck, staticness, structural
representation checks, Go differential oracles, selected tests, extension
checks, and performance gates applicable to generated code. Passing those
gates derives `accepted-manual`; it does not require a promotion file or mutate
the generated baseline marker to resemble the manual body.

An explicit prune command supports `--dry-run` and a separate apply mode for
unreachable/orphaned manual units. It deletes only identities proven
unreachable against the complete current graph and current workspace hash.
Automatic regeneration never prunes manual source.

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
- every manual body resolves to one current typed AST body and one derived
  baseline/hash/ownership record;
- every extension seam resolves to one typed assembly decision;
- no unit has multiple ownership classes;
- generated source contains no text patches or runtime owner selection; and
- unresolved external, stale manual, missing extension, and unimplemented core
  closures are reported separately.
