# Normative Testing And Acceptance Contract

Status: normative. This document defines the evidence required to accept
GoToTS and a GoToTS-generated TSTS. A green later-stage suite never substitutes
for a missing earlier-stage proof.

## Test Principles

- Never weaken, delete, skip, or broadly refresh a test to admit a translator
  defect.
- Compare Go language behavior with the pinned Go toolchain and compiler product
  behavior with pinned TS-Go.
- Compute coverage from identity-bearing source and test ledgers, not filenames,
  discovery conventions, or remembered counts.
- Keep LSP and fourslash source, wrappers, fixtures, baselines, tests,
  dependencies, and performance workloads outside every product denominator.
- Treat imported library behavior as an external-emulation responsibility;
  test GoToTS's exact generated stub contract independently.
- Publish deterministic failure artifacts with enough evidence to reproduce the
  failing source, typed decision, generated output, and oracle difference.

Required tests must not use skip, todo, focus/only, or environment-dependent
success. An unavailable required toolchain blocks the gate. Optional developer
probes are not included in acceptance counts.

## Gate Ordering

Run gates in this order. A failure stops publication of authoritative output:

1. repository and specification policy;
2. pin, toolchain, source-tree, profile, and input attestation;
3. typed census and exact disposition coverage;
4. declaration and external-contract verification;
5. body semantic IR and ownership verification;
6. representation-plan verification;
7. lowering and generated-artifact verification;
8. deterministic regeneration;
9. Go semantic differential oracles;
10. generated TypeScript strict/static validation;
11. no-extension TS-Go differential validation;
12. TSTS extension and product validation;
13. complete selected compiler corpus;
14. proof projects and common downstream projects;
15. self-compilation and native-target probes;
16. performance acceptance;
17. upgrade-repeatability proof.

## Developer Checkpoints

Before every commit:

```sh
test -z "$(gofmt -l .)"
go vet ./...
go test -count=1 ./...
git diff --check
```

Before every push:

```sh
go test -count=1 -race ./...
```

The repository must expose one documented full-gate command before translation
scales beyond the vertical architecture slice. That command runs every
applicable acceptance layer, writes a machine-readable report, and exits
nonzero for missing counts, unknown dispositions, skips, timeouts, crashes, or
failed gates.

## Repository And Specification Gates

Tests enforce canonical spec presence, timeless language, file-size and
semantic decomposition, ESM/static-output policy, branch-independent generated
inputs, and absence of source-name semantic rules. Machine-enforceable spec
requirements are not left as prose only.

## Input And Census Fixtures

Fixture repositories prove at least:

- exact source revision, tree, module, toolchain, environment, and build profile;
- bidirectional reconciliation of tracked and compiler-selected inputs;
- ignored, build-tagged, generated, embedded, non-Go, nested-module, tool, test,
  hard-excluded, and unselected files;
- clean and tampered trees, local replacements, submodules, symlinks, and path
  overlap;
- in-package and black-box test identity;
- production, test, and excluded dependency reachability;
- concurrent and interrupted immutable publication;
- manifest verification and byte-identical clean runs.

Every supported or rejected profile/input class has a positive and negative
fixture. The fixture matrix is generated from the profile schema so a field
cannot be added without a disposition.

## Completeness Ledgers

The authoritative run publishes identity-bearing ledgers for:

- every tracked source-bearing unit and its disposition;
- every selected package, file, declaration, signature, constraint, constant,
  body, statement, expression, typed operation, directive, and initialization;
- every representation requirement and selection;
- every external package/object/type/member obligation;
- every generated file, declaration, body, helper, source map, and owner;
- every manual body and product extension seam;
- every in-scope upstream Go test and TSTS product test;
- every unsupported or excluded item and its exact reason.

The union equals the declared source universe and required test universe. The
categories are disjoint where ownership requires it. Aggregates are recomputed
from records. Unknown, duplicate, orphaned, or missing identities block output.

## Independent Declaration Verification

A verifier independent from the emitter compares pinned Go evidence with
generated TypeScript declarations. It covers functions, methods, receivers,
parameters, named and multiple results, variadics, types, aliases, structs,
fields, tags where relevant, interfaces, embedded members, constraints,
instantiations, constants, variables, arrays, slices, maps, pointers,
channels, functions, and external named types.

The verifier also checks generated TS-Go schema kinds, node fields, unions,
factories, predicates, casts, and visitor/factory order against the pinned
schema. Missing, extra, reordered, or hand-maintained schema elements fail.

## Body And Representation Verification

Every selected body is generated or explicitly manual. Tests reject bodies
with two owners, missing bodies, orphan manual implementations, stale body
hashes, and unrepresented typed operations.

For each representation class, an independent validator checks that selected
storage satisfies all observed requirements. Adversarial fixtures exercise
requirements near the proof boundary. For example, a native-array slice proof
is invalidated by an added capacity read, aliased reslice, element address, or
append whose backing reuse is observable.

The conservative and optimized lowerings are compared on generated small
programs where both are available. This is differential evidence, not a runtime
fallback in product output.

## Go Semantic Oracles

Small typed fixtures are compiled and executed with the pinned Go toolchain,
translated, executed as generated TypeScript, and compared. Compare return
values, writes, stdout/stderr where relevant, panic category and ordering,
observable state, and termination.

Required semantic families include:

- evaluation order, short declarations, shadowing, multiple assignment, and
  control flow;
- integer widths, constants, conversions, shifts, floating point, and complex
  values present in the corpus;
- strings as UTF-8 bytes/runes and JS strings only under proven equivalence;
- arrays, slices, nil/empty, capacity, append, reslice, copy, aliasing, and
  element addresses;
- maps, nil behavior, key equality/hash, comma-ok, iteration, and deletion;
- pointers, field/element addresses, escapes, equality, and mutation;
- struct copy/zero behavior and embedded fields;
- interfaces, typed nil, dynamic identity, assertions, switches, and method
  dispatch;
- functions, closures, receivers, variadics, multiple results, generics, and
  method values;
- defer, panic/recover, initialization, goroutines, channels, and select forms
  present in the corpus.

Property-based generation composes supported semantic domains and shrinks
failures. Generated programs stay within the declared supported subset; every
generated case records its seed and canonical minimized reproduction.

## Generated TypeScript Gates

The complete generated tree must:

- parse and typecheck under strict selected TypeScript options;
- use ESM with explicit `.js` imports;
- contain no unresolved import or placeholder required by the product run;
- pass structural checks for `any`, unproven casts, reflection, dynamic
  property selection, runtime source-name dispatch, CommonJS, triple-slash
  references, forbidden globals, and substitute compiler-state tables;
- conform to the selected Tsonic-compilable grammar;
- have deterministic formatting, imports, names, layout, source maps, and
  provenance;
- respect generated/manual/extension/emulation ownership boundaries.

Scanners use parsed TypeScript structure and symbol evidence where practical;
regex alone never establishes semantic compliance.

## Deterministic Regeneration

Run two generations from independently empty destinations with identical
attested inputs. Compare file inventory and bytes for generated source,
language ABI, external contracts, source maps, manifests, declaration/body
proofs, representation plans, seam reports, manual-body reports, unsupported
reports, and test ledgers.

Then vary ambient working directory, temporary directory, home directory, file
enumeration order, and permitted process environment. Output identity remains
unchanged. Machine evidence may differ only in its separately identified
non-deterministic evidence file.

## No-Extension TS-Go Differential

Build generated TSTS with the product extension host absent. Run the same
project set through pinned TS-Go and generated TSTS. Compare:

- exit status and crash/hang/timeout behavior;
- diagnostics code, category, message arguments/rendering, file, span, order,
  related information, and deduplication;
- emitted file inventory, bytes, source maps, and build metadata where the
  contract requires exactness;
- module resolution and source-file identity;
- parser/scanner AST and token behavior through schema-level observables;
- incremental state and update behavior for selected workloads.

Normalize only explicitly documented host differences. Each normalization has
an identity, rationale, owner, and focused test. A blanket golden refresh is
not a differential resolution.

## TSTS Product Extension Gates

With the complete product layer enabled, validation proves:

- every typed seam resolves to one expected semantic operation and cardinality;
- every required selected symbol, declaration, signature, type argument,
  result type, or flow value is carried from the selecting operation;
- extension code does not re-enter checker queries to reconstruct selected
  evidence;
- compiler-state augmentation has one central typed owner;
- lifecycle finalization, fact conflicts, diagnostics, providers, virtual
  modules, module slices, source profiles, and embedding are order-independent;
- absent extension hosts preserve the no-extension differential result;
- unchanged manual product modules strictly compile against the generated
  extension-core contract;
- a moved or incompatible upstream seam produces a categorized blocking delta.

All TSTS extension/provider tests receive explicit ledger dispositions. A file
layout change is not permission to drop a product behavior test.

## Upstream And Product Test Ledger

Every in-scope TS-Go `_test.go` test function, benchmark, fuzz target, fixture,
and testdata dependency receives one exact disposition:

- mechanically translated and run;
- represented by a named Go differential oracle;
- covered by a named compiler-corpus case with equivalent assertions;
- product-owned TSTS test;
- explicit manual test implementation;
- excluded by the canonical LSP/fourslash scope.

File-level presence does not establish case-level coverage. Imports of
`testing` or assertion packages create external contracts; they do not remove
an owned language test from the ledger.

## Compiler Corpus

The runner discovers the complete selected compiler, conformance, project,
transpile, and applicable unit corpus mechanically. It publishes discovered,
scheduled, executed, passed, failed, skipped, timed-out, and missing-count
totals. Every discovered case is scheduled exactly once unless its explicit
profile disposition says otherwise.

LSP and fourslash remain outside discovery and denominators. Historical labels
such as “15k suite” are orientation only; the run report's exact inventory is
authoritative.

Compare TS-Go and generated TSTS diagnostics and output. Differences are
categorized as translator defect, product-extension difference, external
emulation defect, accepted host normalization, or upstream issue. Every
accepted difference has a focused regression and owner.

## Downstream And Self-Compilation Gates

After focused and corpus gates:

1. install the exact generated TSTS package into clean proof projects;
2. run proof-is-in-the-pudding and representative common projects without
   sibling artifact fallback;
3. run Tsonic host/C# integration and applicable Rust integration;
4. compile representative generated core packages through C# and Rust early;
5. expand to compiling complete generated TSTS with Tsonic;
6. compile Tsonic and its targets with the resulting compiler as product
   support becomes available.

Each stage records exact revisions, package hashes, commands, artifacts, test
counts, and runtime results. Generated code that passes Node but depends on
dynamic behavior unavailable to native targets fails the static contract.

## Performance Gate

Run the complete protocol in `performance-and-representation.md` only after
correctness gates pass. Performance evidence never changes semantic expected
results or source/test denominators.

## Upgrade Repeatability

Prove the upgrade mechanism with at least two attested TS-Go revisions:

1. generate and validate the accepted pin;
2. census the second revision;
3. publish exact file, declaration, operation, external, test, and seam deltas;
4. verify every unrecognized idiom blocks before output publication;
5. add generic support or an explicit valid disposition;
6. regenerate without reading an existing generated tree;
7. repeat all applicable correctness, product, determinism, and performance
   gates.

The test demonstrates that upstream idiom growth is detected mechanically and
resolved through semantic abstractions rather than source-site patches.

## Failure Evidence

Every failure bundle contains:

- input/source/tool revisions and profile hash;
- failing gate and stable diagnostic code;
- Go source span and canonical declaration/operation identity;
- typed decision, representation plan, and lowering fragment;
- generated TypeScript and source-map location when output exists;
- expected and actual oracle result;
- minimal reproduction or shrink seed;
- stdout/stderr, exit status, timeout, and resource evidence;
- manifest hashes needed to reproduce the run.

Failure bundles do not contain machine-dependent data in deterministic identity
files and never require a dirty sibling repository.

## Completion Rule

GoToTS is complete for a selected TS-Go pin only when every required gate is
green, every source and test identity has exactly one valid disposition, every
reachable external product obligation is implemented for the integrated TSTS
run, two clean generations are byte-identical, no required test is skipped or
missing, no unsupported construct reaches output, and the performance contract
passes.

Intermediate census, declaration, vertical-slice, or focused-test milestones
are progress evidence, not completion.
