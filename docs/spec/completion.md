# Runtime, Manual Completion, And Product Assembly

## Ownership Classes

Language runtime, Go standard library, external environment, manual bodies,
and customer extensions are distinct owners:

| Owner | Responsibility | Must not own |
|---|---|---|
| GoToTS compiler | Go language semantics and generated source | host/library behavior |
| `runtime/` | minimal reusable target machinery for language semantics | ordinary source logic or standard-library APIs |
| `gostdlib/` | manually completed Go standard-library behavior | compiler lowering decisions |
| external contract | unavailable/native/host behavior | source-available Go code |
| manual unit | one complete declaration/body implementation | signature or semantic redefinition |
| extension | separately owned customer product behavior | textual patches or semantic rediscovery |

All owners enter one typed implementation and reachability graph. Ownership is
never inferred from a directory name alone.

## Minimal Language Runtime

The runtime contains only behavior that ordinary direct TypeScript cannot
represent exactly and economically, such as selected-width integer operations,
stable scalar locations, slice backing semantics, structural map keys, Go panic
transport, channel scheduling, or interface metadata when proven necessary.

Before adding runtime machinery, the plan records:

- the exact Go observation direct TypeScript fails to preserve;
- the smallest counterexample;
- all simpler rejected representations;
- affected definitions and reference counts;
- generated-size, typecheck, memory, and runtime cost; and
- differential and mutation proof.

Runtime operations are statically typed and selected by plan. There is no
universal value, operation registry, reflection layer, name-based dispatch, or
payload recovery from `any`/`unknown`. A local need cannot justify a mechanism
on every value or call.

## Manually Ported Go Standard Library

GoToTS does not hand-maintain a parallel description of the standard-library
API. For the selected Go toolchain it selects the authoritative `go list std`
package set, loads those declarations from that toolchain's `GOROOT`, assigns
canonical standard-library identities, and generates strict TypeScript
declarations and typed throwing placeholders into one reusable `gostdlib`
workspace. A module-less package is not rejected or guessed to be standard:
`std`-set membership plus the toolchain's `Standard` and `Goroot` facts
distinguish standard library, other toolchain source, and invalid input.

Behavior is manually implemented against those generated contracts. Generated
and manual units may coexist in each source-mirrored standard-library file.
Applications import canonical `gostdlib` modules; a standard-library
definition is not copied into every generated application.
Standard-library declaration and type source participates in ordinary semantic
analysis, but standard-library provenance assigns its executable bodies to the
manual `gostdlib` contract rather than silently attempting ordinary automatic
body lowering. Each selected body therefore has an explicit placeholder,
accepted manual implementation, or blocking disposition.

Example source declaration:

```go
package os

func ReadFile(name string) ([]byte, error)
```

Initial generated unit:

```ts
/**
 * @gotots-generated
 * source="std:os.ReadFile"
 * implementation="std:os.ReadFile/default"
 * contract-sha256="..."
 * body-sha256="..."
 */
export function ReadFile(
  name: string,
): readonly [GoSlice<GoByte>, GoError | undefined] {
  throw new Error("GOTOTS_STDLIB_UNIMPLEMENTED:std:os.ReadFile/default");
}
```

A developer edits only the body:

```ts
/**
 * @gotots-generated
 * source="std:os.ReadFile"
 * implementation="std:os.ReadFile/default"
 * contract-sha256="..."
 * body-sha256="old-generated-hash"
 */
export function ReadFile(
  name: string,
): readonly [GoSlice<GoByte>, GoError | undefined] {
  return nodeReadFile(name);
}

function nodeReadFile(
  name: string,
): readonly [GoSlice<GoByte>, GoError | undefined] {
  // Manual helper: no generated marker and no registration file.
  throw new Error("example implementation omitted");
}
```

The changed post-format body hash makes `ReadFile` manual. The headerless
helper is manual. Typed AST references create the edge between them.

Each manually implemented standard-library operation has focused Go-versus-TS
contract tests. Package-level tests cover interactions, errors, platform
behavior, initialization, and concurrency. Publication selects only reachable
standard-library declarations and their transitive type/runtime closure, while
canonical source ownership remains singular.

A Go toolchain upgrade regenerates declarations and reports signature,
availability, and reachability changes. It never silently adapts an old manual
body through overloads or wrappers.

## Package Routing And Physical Layout

Relative output layout is fixed even when the owning roots are configured at
different physical locations:

```text
<editable-product>/src/<module-import-path>/<go-source-file>.ts
<gostdlib-root>/src/<standard-import-path>/<go-source-file>.ts
<runtime-root>/src/<semantic-family>.ts
<external-root>/<contract-id>/src/<contract-module>.ts

<machine-root>/baselines/<generation>/...   generated-only, immutable
<machine-root>/staging/<run>/...            fresh generation plus overlay
<machine-root>/evidence/<generation>/...    identities, hashes, and graphs
<publication-root>/<generation>/...         reachable, immutable product
```

For example:

```text
$GOROOT/src/os/file.go
  -> <gostdlib-root>/src/os/file.ts

github.com/acme/log/log.go from a module dependency
  -> <editable-product>/src/github.com/acme/log/log.ts
```

Workspace modules, source-available module dependencies, and selected
source-available toolchain packages use the ordinary source-mirrored product
tree. Standard-library packages use the canonical reusable `gostdlib` root.
Unavailable-source contracts own modules under the external root. A host-owned
operation declared inside a source-available package keeps its declaration in
the source-mirrored module; only its implementation obligation is externally
owned.

The routing decision consumes typed package provenance and implementation
ownership. It is never inferred from the output directory. There are no
`.generated.ts`/`.manual.ts` pairs, per-function stub files, or attachment JSON.
All selected declarations from one Go source file share its mirrored TypeScript
file by default, while generated and manual ownership remains per declaration
or body.

## Generated And Manual Units In One File

File-level generated ownership is forbidden. Ownership is structural and
applies to the smallest independently replaceable unit:

- function or method body;
- constructor or accessor body;
- package/class/static initializer body;
- function literal body, deepest first;
- wholly generated declaration without a body; or
- wholly manual declaration.

Every generated unit has a mandatory marker containing source identity,
implementation identity, generated-contract hash, and generated-body hash.
Every generated executable body has its own body hash. A generated class shell
and its generated methods are independently owned; editing one method does not
freeze the rest of the class.

A unit is classified as:

- **generated** when its marker is valid and its canonical current body hash
  equals the immutable generated baseline hash;
- **manual override** when the marker is valid but the current body hash
  differs;
- **manual declaration** when it has no generated marker; or
- **invalid** when markers, identities, hashes, nesting, or contracts are
  malformed, duplicated, forged, or ambiguous.

The body hash is computed from the deterministically formatted body AST after
formatting, not raw source bytes. Formatting-only edits therefore do not claim
manual ownership. Contract hashes cover the generated signature, modifiers,
type parameters, fields, heritage, and other non-body shape. Editing a
generated contract is a conflict, not a manual override. A wholly manual
declaration may define its own complete contract because it has no generated
marker.

The compiler writes an immutable machine baseline and ledgers automatically.
These prevent a user from editing both a body and its marker to impersonate
generated output. They are compiler evidence, not user-authored attachment
metadata. Writing TypeScript is the only action required to implement a
placeholder or add a helper.

## Placeholder Contract

When analysis identifies a body but automatic lowering is unavailable, manual
completion mode may emit its exact generated declaration with one stable
throwing body:

```ts
/** @gotots-generated ... body-sha256="..." */
export function Difficult(value: Value): Result {
  throw new Error("GOTOTS_MANUAL_UNIMPLEMENTED:<ImplementationID>");
}
```

External and standard-library placeholders use distinct stable error families.
Replacing the body converts it to a manual override automatically. A reachable
placeholder blocks certification and publication even if strict TypeScript
typechecking succeeds.

## Regeneration Algorithm

Regeneration is a structural rebuild, not an in-place patch:

1. Fingerprint the new Go workspace, toolchain, compiler, policies, standard
   library, external contracts, extensions, prior immutable baseline, and
   editable mixed workspace.
2. Parse and strictly resolve the prior editable TypeScript.
3. Verify every generated marker against the immutable prior baseline.
4. Extract manual overrides and headerless declarations as typed AST units.
5. Generate a complete new automatic baseline into an empty staging root.
6. Discard all prior generated units, imports, aliases, helpers, and modules.
7. Join manual overrides to fresh implementation identities and contract
   shapes; never join by spelling or line number.
8. Overlay compatible manual bodies/declarations structurally into the fresh
   AST.
9. Recompute imports and the complete implementation graph from the combined
   AST and sealed semantic plan.
10. Classify manual units as current, stale, missing target, incompatible,
    orphaned, unreachable, automatically lowerable, or invalid.
11. Run reachability, strict verification, differential tests, and all
    required gates.
12. Publish the immutable staged root atomically or leave the current product
    untouched.

If a source signature changes, the prior manual body receives a structural
contract diff and blocks assembly. GoToTS does not inject defaults, create a
compatibility overload, or call the old body through a wrapper.

When an automatically generated implementation becomes available for a
currently manual body, the manual implementation remains authoritative until
an explicit identity-targeted reset. Reset compares the expected current hash
before replacement and has dry-run output.

## Complete Reachability Graph

Reachability is full graph traversal over current implementation identities,
not file presence, source imports, name scanning, or a generated-only graph.

Node classes include:

- Go declarations, initializers, automatic implementations, and
  specializations;
- manual declarations and bodies;
- classes, methods, fields with initializers, closures, and callbacks;
- runtime and standard-library definitions;
- external adapters and implementations;
- extension modules and seam contributions; and
- published modules and re-exports.

Edge classes include:

- direct calls, constructors, field/static initializers, imports, and
  re-exports;
- exact method selections, method/function values, closures, and callbacks;
- finite interface targets, conversion adapters, assertions, and RTTI;
- generic instantiation and specialization;
- class heritage and implemented contracts;
- package initialization, blank imports, and registration effects;
- runtime/standard-library/external calls; and
- extension anchors, ordering, initialization, and public assembly.

Go semantic edges come from the sealed plan. Manual TypeScript edges come from
the pinned TypeScript parser, resolver, and TypeChecker over the reconciled AST.
No regex or raw import text is authoritative. Genuinely callback-driven
external behavior contributes edges through its typed contract. Dynamic name
lookup or reflection without an explicit finite contract is unsupported.

Roots are explicit product executables, public API surface, selected tests,
standard-library/runtime entry requirements, and extension assembly roots.
Manual history never creates a root merely to keep code alive.

The graph traversal produces, for every retained or rejected node, at least one
root witness or one exact non-reachability explanation. Type-only closure and
runtime closure are reported separately and then combined for module
materialization.

## Reachability And Pruning Example

Version one:

```go
func Entry() int { return OldHelper() }
func OldHelper() int { return 1 }
```

Assume `OldHelper` has a manual TypeScript body. Version two changes `Entry`:

```go
func Entry() int { return Current() }
func Current() int { return 2 }
```

The fresh graph has no root path to the old manual implementation. It is
reported as unreachable. By default its source is left untouched; it is not
copied into the retained publication AST. This remains true when it shares an
editable file with reachable units: structural assembly omits the unreachable
declaration while leaving the editable workspace intact. `gotots manual prune`
first emits a dry-run plan containing the exact nodes, lost root paths, reverse
references, and workspace hash.
`--apply` deletes only if that hash still matches. If another manual helper
calls `OldHelper`, the edge keeps both reachable.

## External Contracts

Source-available Go dependencies are translated normally. An external
contract is required only for unavailable source or behavior that belongs to
the host/platform, including native bindings, selected `unsafe`/`cgo`
operations, process/filesystem/network adapters, or intentional JavaScript
integration.

For every reachable external declaration or operation GoToTS generates:

- exact translated type/signature and canonical obligation identity;
- receiver, value-copy, nil, error, panic, callback, and effect contract;
- imports, generic bindings, and representation adapters;
- one post-format hashed throwing placeholder; and
- no guessed implementation.

Manual implementation uses the same ownership mechanism as application and
standard-library bodies. An obligation record atomically stores identity,
slot/name, type, effects, adapter, and implementation status. Parallel maps or
consumer recomputation are prohibited.

Only reachable obligations block a selected product, but the report separately
shows declaration/type closure and runtime closure. External behavior must be
contract- and differential-tested against its actual provider.

## Customer Extension Architecture

The existing customer-facing extension architecture is a compatibility
requirement and must be retained as the single public extension contract. Its
current observable request, ordering, diagnostics, and assembly behavior must
be frozen in executable compatibility fixtures before the cleanroom cutover.
There is no old/new extension mode.

Internally, the contract is implemented at one generic boundary:

```text
finalized selected semantic evidence
        -> extension selection and ordering
        -> typed extension contribution and graph edges
        -> product assembly
```

Extensions may provide external implementations, initialization, typed
declarations, product modules, or behavior at stable semantic seams. A seam is
identified by canonical declaration/operation identity, phase, typed inputs,
outputs/effects, cardinality, and order—not a source line, textual marker, or
emitted name.

Extensions must not:

- import compiler internals or acceptance-corpus packages;
- re-enter `go/types` or semantic analysis to rediscover missing evidence;
- inspect generated source text or host object shape;
- patch a partial body by string replacement;
- mutate a parallel compiler-state table; or
- bypass ownership, staticness, reachability, or publication gates.

The generated core is reproducible without extensions. Assembly then applies
the selected extension plan and recomputes the same graph. Every anchor must
resolve exactly once unless its declared cardinality says otherwise. A source
upgrade that removes an anchor reports the extension as unresolved or
unreachable; it never silently moves the behavior.

## Completion Commands

The CLI exposes one structural workflow:

- `gotots generate` creates a fresh baseline and completion report;
- `gotots regenerate` reconciles a prior editable workspace;
- `gotots inspect constructs` explains language occurrences and plans;
- `gotots manual status` lists ownership, conflicts, placeholders, and root
  witnesses;
- `gotots manual reset` replaces selected manual bodies with current generated
  implementations using compare-and-swap hashes;
- `gotots manual prune` reports or applies unreachable-manual deletion; and
- `gotots gate` verifies and attests a staged product.

These commands call the same compiler pipeline. None is a compatibility or
recovery path.
