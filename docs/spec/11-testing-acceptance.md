# Testing And Acceptance

## Testing Principles

Tests prove semantic classes, accounting, and assembled product behavior. A
green subset cannot imply complete coverage. Tests are never weakened to match
an implementation defect.

The pinned Go toolchain is the semantic oracle. Models and handwritten expected
values supplement but do not replace Go execution.

## Ordered Gate Families

Gates execute in this order:

1. repository formatting, policy, schema, and specification;
2. input/toolchain/profile attestation;
3. selected scope and dependency closure;
4. census and denominator reconciliation;
5. declaration/signature/type completeness;
6. semantic IR and operation-class completeness;
7. ownership and support-state completeness;
8. fixed-point and representation verification;
9. deterministic staged generation;
10. strict TypeScript and staticness;
11. semantic unit, property, mutation, and differential oracles;
12. generated package and selected translated-test execution;
13. no-extension TypeScript-Go compiler differential;
14. extension and assembled TSTS product validation;
15. selected compiler corpus, proof projects, and common projects;
16. performance;
17. source-update repeatability; and
18. complete selected-product publication.

A later gate cannot compensate for an earlier failure. An unimplemented unit is
a reported blocked product gate, not a failed test and not a skipped test.

## Repository Gates

Required local checks include:

- `gofmt`;
- `go vet ./...`;
- `go test -count=1 ./...`;
- `go test -count=1 -race ./...`;
- `git diff --check`;
- specification manifest/link/language checks; and
- repository line-size checks.

Tests that mutate global process state isolate and restore it. Tests do not
depend on execution order, ambient workspace state, network, or writable source
inputs.

## Census Fixtures

Synthetic Git repositories prove:

- toolchain/source tamper rejection;
- ignored-file injection rejection;
- selected/outside root filtering;
- selected dependency into outside scope rejection;
- build tags and generated-file evidence;
- in-package and black-box package identity where tests are selected;
- external origin and declaration closure;
- duplicate/invalid identity rejection;
- clean-before/after verification;
- path/symlink containment; and
- byte-identical repeated reports.

LSP, fourslash, and editor-service files are absent from the GoToTS test
universe. Scope tests use synthetic roots rather than enumerating those trees.

## Declaration And IR Tests

Each declaration and operation schema has:

- positive parser/frontend fixtures;
- malformed and contradictory evidence fixtures;
- canonical identity tests;
- deterministic serialization tests;
- source-span tests;
- upgrade addition/removal tests; and
- verifier mutation tests.

Every selected body has complete IR or one unimplemented record. A body cannot
be counted in both states.

## Semantic-Class Oracles

Each implemented class has a focused Go fixture and generated TypeScript
execution. The harness compares:

- values and exact types where observable;
- mutation and aliasing;
- panic category, value, and timing;
- evaluation order and trace;
- nil/empty behavior;
- copies and storage identity;
- map presence/equality/hash;
- string bytes and rune behavior;
- interface dynamic type and typed nil;
- generic instantiations;
- defer/recover;
- initialization; and
- permitted concurrency outcomes.

One oracle covers a class, with generated/property cases exploring its type and
operation dimensions. It does not require manually authored tests for every
call site.

Passing a class oracle establishes `semantic-class-validated` evidence only for
the dimensions represented by the oracle and its generators. It does not prove
that every body using the operation has correct evaluation order, surrounding
alias/effect interactions, boundary conversion, package initialization, or
integration behavior. It therefore cannot establish `package-executed`,
`compiler-differential-validated`, or `certified` by implication.

For example, a compound-assignment oracle must include an RHS that mutates the
same storage location:

```go
x := 1
x += func() int { x = 10; return 2 }()
```

Go loads the old value before evaluating the RHS and produces `3`; a lowering
that reloads after the callback produces `12`. A body containing `+=` is
`exposed` when this shared lowering is defective, but is a
`confirmed-defect` only when the mismatch is shown to be observable for that
body or the shared emitted form is unconditionally invalid.

Oracle transport is a versioned canonical tagged event stream, not ordinary
JSON values or ad hoc stdout. It preserves integer width, bigint, float bits
including NaN and signed zero, arbitrary bytes, dynamic defined type, nil and
typed nil, panic payload/category/timing, mutation trace, and equality
relations among storage identities. Run-local pointer IDs are compared by
their observed equivalence graph, never by host address.

For Go-permitted nondeterminism such as map range or select, tests validate the
specified allowed-outcome model and required invariants over deterministic
seeds/schedules. One incidental upstream outcome is not the complete oracle.

## Representation Transition Tests

Tests cover every requirement transition, candidate capability boundary, and
legal conversion:

- direct representation remains selected when stronger observations are absent;
- adding one observation propagates to all aliases/calls;
- unrelated regions remain direct;
- removing or changing an effect invalidates cached plans;
- unknown boundaries select conservative behavior or unimplemented;
- custom output requires its necessity record; and
- generated output contains one representation.

Mutation tests remove graph edges, alter constraints, or forge proof records and
require the independent verifier to fail.

## Property And Metamorphic Testing

Generated test families cover:

- numeric boundaries and conversions;
- nested struct/array copying;
- slice alias/reslice/append/copy/clear;
- map key classes and mutation during range;
- valid and invalid UTF-8 strings;
- pointer alias/equality/lifetime;
- interface assertions/equality/uncomparable panic;
- generic specializations;
- assignment and control-flow evaluation order; and
- defer/panic/concurrency interactions.

Seeds, generators, shrink results, and oracle revisions are recorded.

## Unimplemented Tests

Each unimplemented class has a focused fixture proving:

- recognition is typed and stable;
- diagnostic identity and source evidence are exact;
- all mechanically matching sites receive the same class;
- affected artifact closure is withheld;
- independent supported units remain analyzable; and
- an editable typed placeholder, when emitted, throws the exact unimplemented
  diagnostic and is never classified as retained runnable or product output;
- no approximate output or fabricated return appears.

Implementing the class converts the fixture into a positive differential oracle;
the unimplemented count decreases by exact identity.

## Manual Ownership And Regeneration Tests

Fixtures exercise the complete mixed-source lifecycle:

- generated and manual bodies in one file and one class;
- exact post-format body hashes for functions, methods, constructors,
  accessors, initializers, and nested function literals;
- formatting-only changes normalizing to generated ownership;
- semantic edits becoming manual without JSON or promotion metadata;
- forged, malformed, duplicated, removed, and wrong-identity markers failing;
- immutable-baseline verification preventing marker/body collusion;
- deepest-first nested-body ownership without overlapping owners;
- fresh empty-root generation followed by structural manual overlay;
- no old generated body/import/helper surviving regeneration;
- automatic static dependency and reachability derivation through direct calls,
  function values, callbacks, finite interface/generic targets, init, externals,
  extensions, and tests;
- current, placeholder, stale, missing, unreachable, orphaned,
  automatically-lowerable, and invalid statuses;
- source upgrades that add, change, move, or remove a manual body's Go object;
- explicit reset with current-hash compare-and-swap semantics;
- prune dry-run/apply preserving reachable source and rejecting a stale plan;
  and
- reachable unresolved manual/external placeholders blocking publication.

Regeneration tests prove that manual source is preserved by ownership, not by
copying whole old files. Generated output surrounding a manual body must update
to the new baseline in the same run.

## Selected Go Tests

Selected non-editor Go tests receive one machine disposition:

- translated and executed;
- represented by a stronger semantic differential test with exact ownership
  linkage;
- external-emulation-owned; or
- unimplemented/blocking.

Fixtures, goldens, testdata, subtests, benchmarks, and fuzz targets have
identity-bearing records where selected. A file's presence does not prove its
cases ran.

A replacement test is stronger only with a machine dominance record mapping
the original test identity, setup, dynamic subtests, operations, assertions,
side effects, panic/failure modes, and relevant input domain to superset
evidence. Human assertion that a test is redundant is not a disposition. When
dominance cannot be proved, the original selected test is translated and run
or remains blocking.

LSP/fourslash/editor tests receive no disposition because they are outside the
universe.

## Strict TypeScript Gate

Generated output is parsed and typechecked using the pinned strict TypeScript
configuration. The gate rejects:

- implicit or explicit unsafe `any`;
- unchecked `unknown` recovery;
- unresolved imports;
- CommonJS;
- string-selected static members;
- reflection/dynamic code generation;
- multiple or runtime-selected representation forms for the same region or
  undeclared boundary;
- missing external/manual/extension contracts; and
- generated source not represented in the manifest.

The generated AST and printed source are structurally reconciled.

Strict TypeScript success proves syntax, imports, and declared static types. It
does not prove Go semantics, runtime execution, package completeness, external
behavior, or certification. A withheld package is not covered by this gate
merely because some of its bodies have lowered AST records.

## Staticness Sweep

The staticness gate performs a complete typed-AST sweep across generator-owned
ABI/templates, generated core, generated helpers, external stubs, manual-body
assembly, extension bridges, and selected-product output. Every invocation,
member selection, callback target, interface branch, external binding, and
representation decision receives one machine disposition:

- direct statically typed operation;
- finite typed dynamic Go-semantic state with closed target branches and an
  accepted necessity record; or
- unimplemented/blocking.

The sweep rejects erased `Function`/`unknown[]` calls, string-selected members,
universal operation registries, reflection-like target recovery, typed wrappers
around erased dispatch, and dynamic fallback paths. It also verifies that
whole-product target analysis was attempted before accepting any runtime
discriminant. Suppressions and file-local allowlists cannot satisfy this gate.

## Compiler Differential

With product extensions disabled, generated TypeScript-Go behavior is compared
with the pinned Go executable on selected compiler workloads. Evidence includes:

- diagnostics and ordering;
- emitted files and source maps where deterministic;
- exit status, panic, crash, hang, and timeout;
- project graph and incremental behavior;
- scanner/parser/binder/checker/emitter results; and
- process/resource usage.

Only explicitly owned host normalization is permitted.

## Product Assembly Tests

With extensions enabled, tests prove seam identity, cardinality, order, state
mutation, selected evidence, fact lifecycle, providers, virtual modules,
sessions, source profiles, diagnostics, and embedding behavior.

External emulation bindings and manual bodies typecheck and execute against the
same generated contracts used by the product.

## Coverage Reports

Every accepted run reports:

- selected source files/packages;
- declarations, function/method bodies, function literals, synthetic
  initializers, and unsupported non-body declarations as separate denominators;
- generated, manual, and unimplemented support states;
- every translation evidence stage from `00-authority-scope.md`;
- directly incomplete, transitively withheld, retained runnable, and published
  package counts;
- operation classes and sites;
- operation-bearing body counts and deduplicated exposure unions;
- confirmed-defect, exposed, and certified body counts;
- test entities scheduled/passed/failed/timed out/crashed;
- missing machine counts;
- external resolved/unresolved;
- extension seams assembled/missing;
- emitted/withheld artifacts; and
- representation forms.

Each count is backed by canonical IDs and exact joins. In particular, an
`ir-admitted` or `lowered` percentage cannot be labeled “translated coverage,”
and bodies removed by package withholding cannot appear in emitted, runnable,
typechecked, executed, or certified totals.

No skipped, todo, focused-only, or missing-count test can satisfy complete
selected-product acceptance.

Passing gates 1–10 establishes neither body correctness nor selected-product
completion. No body is reported `certified` until all applicable gates through
18 pass, including generated execution, compiler differential, extension
assembly, representative projects, performance, source-update regeneration,
and complete publication.

## Deterministic Regeneration

Two clean runs compare every generated byte and canonical record. Tests vary
working directory, home/temp roots, allowed environment ordering, and process
layout without changing semantic identity.

The generated baseline is reconstructible without reading an accepted output
tree. Mixed-source assembly additionally proves deterministic reconciliation
from the same attested prior baseline and editable workspace.

## Acceptance Results

Gate status is one of:

- `pass`;
- `fail`; or
- `blocked`.

Blocked names the missing implementation, unimplemented class, or unavailable
required environment. A gate report is successful only when every gate required
for its declared completion level passes.
