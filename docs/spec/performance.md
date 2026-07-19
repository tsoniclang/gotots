# Performance Contract

## Principle

Correctness is never traded for speed. Within exact semantics, generated TSTS
performance and GoToTS generation cost are product requirements.

The representation planner exists to avoid unnecessary carriers, allocations,
copies, conversions, bounds checks, and indirect calls. It cannot remove an
observable Go behavior.

## Performance Shape

Hot generated compiler paths should use:

- ordinary local variables and direct fields;
- native arrays, strings, and maps when proven exact;
- direct calls and closed specializations;
- simple loops;
- scalar-expanded temporary state rather than escaping objects;
- statically absent extension/concurrency machinery when unused; and
- compact deterministic modules.

Custom carriers and runtimes are localized to classes whose necessity records
prove they are unavoidable.

## Required Baselines

Performance reports identify:

1. pinned upstream TypeScript-Go executable;
2. accepted hand-maintained TSTS reference revision;
3. generated TSTS without product extensions; and
4. generated assembled TSTS with selected extensions and emulation.

Each report records source/tool revisions, generated manifest, Node and Go
versions, operating system, architecture, CPU, memory, power mode, harness,
workloads, raw samples, and estimator configuration.

Baseline updates are reviewed independently from optimizations and retain
comparison with the prior accepted baseline.

## Metrics

At minimum measure:

- wall and CPU time;
- peak resident memory;
- allocation count/bytes and garbage-collection time where observable;
- startup and package initialization;
- generated source, source-map, and manifest size;
- GoToTS generation time and peak memory;
- scanner token throughput;
- parser source throughput;
- binder/checker/resolver throughput;
- printer/emitter throughput;
- incremental update latency;
- complete selected corpus time and counts;
- extension-disabled and extension-enabled cost; and
- representation counts and conversions.

## Workloads

### Semantic Microbenchmarks

Measure numeric operations, struct/array copy, slice operations, map key
classes, UTF-8 operations, pointer locations, interface calls/assertions,
generic specialization, defer/panic, and any accepted concurrency runtime.
Every benchmark has a correctness oracle.

### Compiler Macrobenchmarks

Measure scanner, parser, binder, checker, module resolution, emit, incremental
updates, and complete builds on representative small, medium, and large
projects.

### Product Workloads

Measure the selected compiler corpus, selected non-editor TSTS product suite,
proof projects, and common projects. The machine census defines the denominator.

## Measurement Protocol

- Correctness and static gates pass before benchmarks.
- Candidate and baseline run on the same idle machine.
- Order is randomized where safe.
- Macrobenchmarks use at least three warmups and seven measured runs.
- A reported p95 uses at least forty independent measured samples or a
  workload-specific statistically justified larger count; smaller samples omit
  p95 rather than presenting noise as evidence.
- Raw samples are retained.
- Reports include median, meaningful p95, median absolute deviation, and a 95%
  bootstrap confidence interval.
- Outliers require machine-readable causes.
- Unrelated test processes do not run concurrently.

## Default Budget Matrix

Unless an accepted workload-specific budget is stricter, generated
no-extension TSTS is compared with the accepted hand-maintained TSTS revision;
assembled TSTS is compared with generated no-extension TSTS for extension
overhead; GoToTS generation is compared with the prior accepted GoToTS
revision. The upstream Go executable remains a semantic and absolute reference
whose ratio is reported, not an interchangeable default comparator.

Against the applicable comparator:

- product macrobenchmark wall/CPU regression: at most 5%;
- peak resident memory regression: at most 10%;
- startup/initialization regression: at most 5%;
- unexplained generated source/source-map growth: at most 10%; and
- no-extension hot seam overhead: at most 2%.

For sampled metrics, the upper one-sided 95% confidence bound must fit the
budget. Inconclusive evidence triggers bounded additional sampling and then
blocks.

The report states a maximum sample count before measurement begins. Reaching
that bound without a conclusive result blocks; it does not permit unbounded
sampling or comparator switching.

## Representation Evidence

Reports count:

- native versus escalated slices/maps/strings;
- allocated versus scalar-expanded views;
- pointer eliminations/cells/locations;
- copied and elided value operations;
- boxed, union, and devirtualized interfaces;
- shared and specialized generics;
- custom runtime allocations and dispatch;
- bounds checks retained/eliminated; and
- boundary conversions.

A custom mechanism cannot be accepted without an isolated benchmark against
the closest localized exact alternative and the applicable accepted production
baseline. An intentionally inexact host operation is a semantic
counterexample, not a performance comparator.

## Optimization Rules

An optimization record states:

1. the preserved Go behavior;
2. eligible regions;
3. proof predicate;
4. emitted TypeScript;
5. invalidation dependencies;
6. semantic oracle;
7. benchmark result; and
8. allocation, dispatch, and code-size effects.

Optimizations are class-based. Source-name or file-specific fast paths are
forbidden.

Asymptotic complexity is part of correctness/performance review. A translation
cannot turn a selected constant-time operation into input-linear work inside a
hot loop without an accepted class-level exception. Reports also bound and
measure specialization/module growth, generated helper count, largest
representation region, fixed-point iterations, graph memory, and proof-bundle
size. Exceeding a committed bound blocks rather than silently widening regions
or adding wrappers.

## Performance Exceptions

A temporary exception identifies workload, evidence, root cause, owner,
accepted regression, and expiry condition. It cannot waive correctness,
coverage, determinism, or a custom-mechanism necessity proof.

Unimplemented is preferable to introducing a broad slow abstraction solely to
increase translation coverage.
