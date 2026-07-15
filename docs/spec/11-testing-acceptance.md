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
- no behavior-placeholder or approximate output appears.

Implementing the class converts the fixture into a positive differential oracle;
the unimplemented count decreases by exact identity.

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
- declarations and bodies;
- generated, manual, and unimplemented support;
- operation classes and sites;
- test entities scheduled/passed/failed/timed out/crashed;
- missing machine counts;
- external resolved/unresolved;
- extension seams assembled/missing;
- emitted/withheld artifacts; and
- representation forms.

No skipped, todo, focused-only, or missing-count test can satisfy complete
selected-product acceptance.

## Deterministic Regeneration

Two clean runs compare every generated byte and canonical record. Tests vary
working directory, home/temp roots, allowed environment ordering, and process
layout without changing semantic identity.

The generated tree is reconstructible without reading an accepted output tree.

## Acceptance Results

Gate status is one of:

- `pass`;
- `fail`; or
- `blocked`.

Blocked names the missing implementation, unimplemented class, or unavailable
required environment. A gate report is successful only when every gate required
for its declared completion level passes.
