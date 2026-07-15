# Authority, Mission And Scope

## Authority

Requirements are interpreted in this order:

1. the pinned Go language semantics and selected build profile;
2. this complete specification;
3. accepted architecture decisions;
4. commit-bound machine evidence required by this specification;
5. implementation documentation and comments.

An architecture decision chooses an implementation that satisfies the governing
semantics. It cannot waive exact accounting, deterministic input, static output,
fail-closed behavior, or a required acceptance gate.

## Mission

GoToTS translates the complete selected TypeScript-Go compiler corpus into
readable, efficient, strict ESM TypeScript. It must:

- preserve every runtime-observable Go behavior used by the selected program;
- retain complete typed Go semantics in its IR and proof artifacts;
- emit ordinary TypeScript whenever ordinary TypeScript is exact;
- introduce custom runtime machinery only after class-level necessity proof;
- regenerate deterministically from attested inputs;
- survive TypeScript-Go source updates by recognizing and classifying added
  semantic shapes;
- compose separately owned product extensions without patching generated text;
  and
- report incomplete support honestly.

GoToTS is not required to translate arbitrary Go programs. Its implementation
is generic by semantic operation and type class so that a newly selected
TypeScript-Go use either joins an implemented class or blocks as an explicit
unimplemented class.

## Selected Product Universe

The selected universe is defined by the committed project profile and pinned
build configuration. It contains selected TypeScript-Go production packages,
selected tools, and selected non-editor tests.

The following are outside the universe:

- LSP implementation;
- fourslash infrastructure;
- editor-service-only implementation;
- their fixtures, testdata, tests, generated files, and tools.

These roots are completely outside the GoToTS input universe.
Outside-universe roots are filtered before source census. They receive no
per-file or per-declaration disposition. The profile loader verifies that no
selected package depends on an outside-universe package.

Imported packages not owned by the selected universe are external obligations,
not selected source. Standard-library status does not make a package part of
the translator.

Scope classes are mutually exclusive:

- `selected-owned` is a declared selected root or an allowed in-checkout
  dependency reached from one;
- `outside-universe` is below a declared outside root and cannot be reached;
- `unselected` is in the pinned checkout but in neither a selected root nor its
  dependency closure; and
- `external` is a permitted imported package outside selected owned source and
  explicitly excludes every outside-universe root.

An allowed in-checkout import joins `selected-owned`; it is never downgraded to
an external stub. Outside and unselected source receive no semantic census or
translation disposition.

## Product Roots And Reachability

The project profile declares every executable, selected test, product-extension
entry, embedding entry, and externally callable product API root. Every exported
selected declaration is a root unless a committed export-surface contract proves
that its package is not published and names its complete internal callers.

Reachability closes over package initialization, direct calls, method sets,
function values, interface targets, callbacks passed to external code, manual
bodies, extension seams, and external call-in contracts. A finite dynamic target
set requires typed closed-world proof. An unknown call-in or target edge is
conservative for effects and representation and is unimplemented when its
implementation closure cannot be proven.

Every body omitted as unreachable has a machine record naming the roots, graph
revision, exclusion proof, and verifier result. Source location, current CLI
usage, and absence from one test run are never reachability proof.

## Governing Invariants

### Semantic Authority

Typed Go frontend evidence identifies declarations, types, selections,
instantiations, operations, constants, addressability, and build inclusion.
Names and source paths are provenance; they never select semantics.

### Complete Semantic IR

Every selected body is either represented completely in typed semantic IR or
classified unimplemented. The IR describes Go operations rather than emitted
TypeScript syntax.

### Simplest Exact Output

Representation begins from observed semantic requirements, not from a universal
runtime carrier. Constraint propagation selects the least elaborate output that
satisfies all requirements for a closed value-flow region.

### Monotonic Planning

Analysis may add requirements but never discard a requirement to obtain a
simpler result. A region can move only toward a representation that preserves
more observable behavior. Emission starts after deterministic fixed-point
convergence.

### Truth Over Recovery

Missing or contradictory evidence never authorizes guessing, source-name
matching, host-shape probing, unchecked casts, or a best-effort artifact.
Unknown typed behavior is unimplemented or blocking.

### One Static Path

The generated artifact has one representation and implementation per region.
It does not inspect runtime shape to choose an implementation.

### Highest Correct Abstraction

Repeated failures are solved at their shared semantic class. File, package,
function, identifier, or first-reproduction special cases are forbidden unless
the Go semantics themselves make that identity relevant.

### Transactional Publication

Generation occurs in an empty staging root. Every required gate passes before a
single atomic publication. Partial generated trees are not product artifacts.

## Language And Library Boundary

Language obligations include:

- declarations, expressions, statements, evaluation order, initialization,
  method sets, generics, functions, closures, and control flow;
- arrays, slices, maps, strings, pointers, interfaces, channels, and numeric
  semantics;
- `len`, `cap`, `append`, `copy`, `clear`, `delete`, `make`,
  `new`, panic/recover, defer, range, goroutines, channel operations, and
  select; and
- compiler-recognized intrinsics selected by exact frontend object identity.

Imported APIs are library obligations. GoToTS records their exact declaration
closure and emits static contracts. It never recognizes `io`, `strings`,
`sync`, or another library by import spelling to inject behavior.

Frontend-proven compiler intrinsics and directives take precedence over the
ordinary imported-library path. Each selected special construct is one typed
intrinsic semantic class with exact identity, or is explicitly unimplemented.
Package spelling alone never creates intrinsic status.

## Support States

Every selected implementation unit has exactly one support state:

- `generated`: typed IR and lowering are complete;
- `accepted-manual`: typed IR is complete and one reviewed structural body
  supplies the implementation;
- `unimplemented`: semantics are recognized but no accepted implementation
  exists.

External declarations and extension seams use separate ownership states defined
in `08-externals-manual-extensions.md`.

Unimplemented is a valid development result, not product success. It permits
the compiler to continue classifying independent work while withholding every
artifact whose dependency closure reaches that unit.

## Completion Levels

### Specification Complete

The governing semantic classes, planning algorithm, ownership rules, machine
contracts, and acceptance gates are defined without contradictory alternatives.

### Translator Capability Complete

Inputs, census, typed IR, planning, diagnostics, deterministic generation, and
support-state reporting operate over the selected universe. Semantic classes
may remain unimplemented and are reported exactly.

### Selected Product Complete

Every selected reachable body is generated or accepted-manual, every reachable
external obligation is bound, every required extension seam is assembled, and
all correctness, corpus, determinism, and performance gates pass. No reachable
unimplemented state remains.
