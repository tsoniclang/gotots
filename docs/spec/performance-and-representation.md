# Normative Performance And Representation Contract

Status: normative. Semantic equivalence is mandatory and is never traded for
speed. Within that boundary, generated TSTS performance is a product contract,
not an optional cleanup phase.

## Performance Objectives

GoToTS must produce TSTS code that:

- keeps scanner, parser, binder, checker, resolver, and emitter hot paths
  direct, statically bound, and predictably shaped;
- avoids representation objects, copying, allocation, bounds checks, and
  indirect calls whenever a complete proof permits their removal;
- preserves a conservative exact representation whenever proof is incomplete;
- runs efficiently as JavaScript and remains suitable for Tsonic compilation
  to C# and Rust;
- does not improve a benchmark by weakening semantics, skipping work, changing
  diagnostics, narrowing the corpus, or moving cost outside the measurement.

There is one Go semantic model. Representation planning selects a static
implementation from that model; runtime fallback between representations is
not part of generated code.

## Representation Proof

Every nonconservative representation selection records:

1. source type and canonical value-flow identity;
2. operations, aliases, escapes, calls, mutations, and effects considered;
3. semantic requirements that are observable;
4. requirements proven absent;
5. selected representation and generated helpers;
6. invalidation dependencies;
7. focused semantic oracle and performance evidence.

Examples:

- A slice may become a native array only when nil, capacity, backing identity,
  aliased reslicing, append reuse, and element address are unobservable.
- A nonescaping pointer may be elided only when all reads and writes can be
  statically rewritten to the same storage.
- An interface call may be direct only when the target set is closed and
  dynamic type identity, typed nil, assertions, and switches are unobservable.
- An integer operation may omit wrapping only when its result is proven in
  range at every observable boundary.

The verifier independently recomputes or checks each proof domain. A planner's
unsupported assertion is not evidence.

## Forbidden Performance Mechanisms

Generated code must not obtain speed through:

- source-name, file-name, or package-API special cases;
- reflective field or method lookup;
- dynamic property names for statically selected members;
- `eval`, generated functions, proxies, or runtime source tables;
- universal operation dictionaries for concrete closed types;
- hidden representation tags used to choose an implementation at runtime;
- unchecked side tables for compiler-owned state;
- removal of panic, bounds, nil, copy, overflow, or evaluation-order behavior
  without proof;
- a separate fast implementation whose semantics differ from the verified
  implementation.

External package performance belongs to the external emulation implementation.
GoToTS keeps calls static but never recognizes `strings`, `io`, `sync`, or any
other imported API to substitute behavior.

## Required Baselines

Performance evidence records four independently identified products:

1. pinned upstream TS-Go executable;
2. accepted TSTS reference revision;
3. generated TSTS with product extensions disabled;
4. generated TSTS with the complete TSTS product extension set enabled.

Each baseline records source and tool revisions, generated manifest, Node and Go
versions, operating system, architecture, CPU model, logical/physical cores,
memory, power mode, process limits, benchmark inventory, and harness revision.
Machine-local paths and timestamps stay outside deterministic benchmark
identity.

No performance acceptance is published until a checked-in baseline manifest and
workload inventory exist. A baseline update is reviewed separately from a
translator optimization and retains the prior evidence.

## Required Metrics

At minimum, collect:

- wall and CPU time;
- peak resident memory;
- allocation count/bytes and garbage-collection time where available;
- startup and module-initialization time;
- generated TypeScript bytes, files, declarations, and source-map bytes;
- GoToTS generation time and peak memory;
- scanner token throughput;
- parser source-byte/file throughput;
- binder and checker throughput;
- module-resolution and filesystem throughput;
- printer/emitter throughput;
- incremental update latency where selected TS-Go supports it;
- full selected corpus wall time, task count, case count, failure count, and
  peak memory;
- no-extension and extension-enabled seam costs;
- representative C# and Rust generated-code size and runtime evidence.

Representation reports include counts for native arrays, full slice views,
pointer cells, elided pointers, copied value structs, map key strategies,
boxed/devirtualized interfaces, indirect calls, emitted/eliminated bounds
checks, integer conversions, generic specializations, and external boundaries.

## Workloads

### Semantic Microbenchmarks

Measure slice range/index/append/reslice/copy, maps by key class, struct copy and
zeroing, pointer/address operations, interface calls and assertions, UTF-8
scanning, integer conversions, multiple results, defer/panic, channels, and
extension bridge operations. Every benchmark has a semantic oracle so a faster
wrong result cannot pass.

### Compiler Macrobenchmarks

Measure scanner, parser, binder, checker, resolver, printer, emit, incremental
update, and complete command-line builds on representative small, medium, and
large projects. Include projects that stress generic types, declaration emit,
module graphs, JavaScript checking, and diagnostics.

### Product Workloads

Measure the complete selected non-LSP/non-fourslash compiler corpus, the full
TSTS suite, Tsonic integration, proof projects, common real projects, and
self-compilation when available. The machine-generated inventory is the
denominator; a hardcoded historical count is not.

## Measurement Protocol

- Run correctness and static-output gates before benchmarks.
- Use the same clean generated artifact for all repetitions of one candidate.
- Run candidate and baseline on the same machine and pinned harness.
- Randomize candidate/baseline order where the harness permits it.
- Use at least three warmup runs and seven measured runs for macrobenchmarks.
- Use enough microbenchmark iterations to reach a stable confidence interval;
  record the selected count and raw samples.
- Report median, p95 where meaningful, median absolute deviation, and a 95%
  bootstrap confidence interval for the relative difference.
- Record every sample, timeout, crash, and outlier decision. Outliers are not
  removed without a machine-readable reason.
- Do not run unrelated test processes concurrently with benchmark processes.

## Default Regression Gates

Workload-specific checked-in budgets may be stricter. Until then, a candidate
blocks acceptance when any of these holds:

- a product macrobenchmark median regresses by more than 5% and the 95%
  confidence interval excludes zero;
- peak resident memory regresses by more than 10%;
- startup or package initialization regresses by more than 5%;
- generated source or source-map size grows by more than 10% without a
  source-corpus or semantic-obligation explanation;
- a hot no-extension seam path regresses by more than 2% outside measurement
  noise;
- specialization growth is unbounded with source size or instantiation count;
- representative generated output cannot compile through a selected Tsonic
  native target;
- any correctness, corpus, staticness, or determinism gate changes outcome.

A reviewed performance exception identifies the workload, evidence, root cause,
owner, expiry condition, and follow-up gate. It never changes the correctness
denominator or silently broadens a budget.

## TSTS-Specific Hot Paths

The performance plan explicitly covers:

- scanner character and token loops;
- parser lookahead, speculation, node creation, and pooled parser lifecycle;
- binder symbol/declaration/locals mutation;
- checker type, symbol, signature, relation, flow, and instantiation caches;
- module resolution and source-file loading;
- diagnostic creation, sorting, and deduplication;
- printer traversal and emit buffers;
- extension-host lookup, observation dispatch, request construction, selected
  evidence materialization, fact publication, and provider virtual modules.

No-extension TSTS must stay on the exact TS-Go semantic path with a cheap static
absence check. Extension-enabled paths use direct generated bridge calls and
typed requests. A universal reflective dispatcher is not permitted.

## Native-Target Shape

Performance review examines generated JavaScript behavior and the static shape
seen by C# and Rust backends. Generated core uses direct properties, direct
calls, closed generics where proven, explicit integer widths, stable classes or
records, ordinary loops, and a small versioned language ABI. It does not depend
on JavaScript prototype mutation, host globals, property reflection, or dynamic
code generation.

Representative generated packages compile through both target families before
the translator scales across the complete corpus. Final acceptance expands the
probe to full TSTS/Tsonic compilation as the target products support it.

## Optimization Review Record

Every accepted optimization answers:

1. Which Go behavior is preserved?
2. Which source flows receive the optimization?
3. Which proof enables it?
4. What TypeScript is emitted?
5. Which invalidation causes conservative replanning?
6. Which semantic oracle covers it?
7. Which benchmark demonstrates value?
8. What allocation, dispatch, code-size, and native-target effects remain?

An optimization without this record remains outside authoritative generation.
