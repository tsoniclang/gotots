# GoToTS Governing Specification

## Authority

This directory is the only normative GoToTS specification. Its six Markdown
files form one contract. Requirements are interpreted in this order:

1. the selected Go language version and build semantics;
2. this specification;
3. accepted architecture decisions that satisfy this specification;
4. revision-bound machine evidence; and
5. implementation comments.

Historical plans, reviews, generated reports, and non-governing implementations are
evidence only. They cannot override this contract. There is no compatibility
specification and no second implementation path.

The terms **must** and **must not** are requirements. **Should** means required
unless a reviewed, strictly stronger mechanism proves every affected
requirement. Examples are normative in semantic shape; incidental identifier
spelling is illustrative.

## Mission

GoToTS is a general Go-to-TypeScript compiler. Given an arbitrary valid Go
workspace, a selected Go toolchain/build configuration, roots, and explicit
environment contracts, it must produce deterministic, readable, strict ESM
TypeScript with Go-equivalent observable behavior.

Generated core TypeScript is also a Tsonic compilation input. It must remain in
the statically compilable Tsonic subset; JavaScript dynamic-invocation or
prototype mechanisms are not an intermediate escape hatch. The final bootstrap
includes translating GoToTS itself and compiling that generated TypeScript with
Tsonic.

Microsoft's `typescript-go` repository is the first large acceptance corpus.
It is not a language authority, compiler dependency, privileged profile, or
source of lowering rules. A fix discovered through that corpus must be reduced
to a project-independent Go construct or semantic operation before entering
the compiler.

GoToTS is complete only when the selected language constructs and environment
contracts are implemented and all applicable gates pass. During development,
an unsupported construct may produce an explicit typed placeholder in manual
completion mode; it must never be reported as translated or publishable.

## Central Contract

```text
selected Go workspace and toolchain
        -> coherent resolved package/source-unit universe
        -> explicit environment/provider evidence-depth selection
        -> complete selected construct and context inventory
        -> typed, target-independent Go semantic model
        -> sealed whole-program facts
        -> one immutable TypeScript representation plan
        -> typed TypeScript AST
        -> generated/manual reconciliation
        -> complete dependency and reachability graph
        -> strict verification and atomic publication
```

The governing equation is:

```text
complex compile-time reasoning + one owner per semantic fact
    = simple, direct, source-shaped TypeScript
```

Runtime dictionaries, repeated aliases, per-call implementer switches, hidden
operation arguments, wrappers, casts, or duplicated definitions are not
acceptable substitutes for compile-time analysis.

## Scope And Boundaries

The compiler owns:

- Go syntax, typing, evaluation order, storage, copying, nil, panic, defer,
  initialization, interfaces, generics, concurrency, and built-in operations;
- target-independent semantic classification and whole-program analysis;
- one exact TypeScript representation for each selected semantic region;
- deterministic typed output, provenance, and source maps;
- generated/manual ownership, regeneration, and full-graph reachability; and
- proof that generated behavior and architecture satisfy this specification.

The compiler does not invent host behavior. The following have explicit
owners:

- `runtime/` implements the smallest reusable machinery needed for Go language
  semantics that ordinary TypeScript cannot express directly;
- `gostdlib/` is the reusable, manually completed TypeScript implementation of
  the selected Go standard-library API, with signatures and placeholders
  generated from that toolchain;
- external contracts own unavailable source, host, native, `cgo`, `unsafe`, or
  platform operations; and
- the existing customer-facing extension architecture owns separately
  supplied product behavior through finalized typed evidence.

All ordinary imported packages have the same Go language semantics. Package
provenance is a separate resolved fact used for identity, output routing, and
implementation ownership; it never changes construct interpretation.
Source-available third-party dependencies are ordinary selected Go source, not
external stubs merely because they are outside the root module. Standard-library
membership comes only from the selected toolchain's package metadata, never
from import-path spelling or a filesystem-prefix test.

## Binding Vocabulary

- **Go construct:** a syntactic form expressible by the selected Go grammar,
  such as assignment, call, declaration, receive, type assertion, or `defer`.
- **semantic variant:** a construct after contextual typing resolves its
  meaning, such as one-result map lookup versus comma-ok lookup.
- **implicit operation:** Go behavior not represented by a dedicated syntax
  node, such as value copying, zero construction, method promotion, boxing, or
  package initialization.
- **source identity:** stable identity for a package, declaration, binding,
  type, construct occurrence, or source span.
- **package provenance:** the toolchain-resolved class `workspace-module`,
  `module-dependency`, `standard-library`, `toolchain-package`, or
  `language-pseudo`; it is not an implementation policy.
- **source acquisition:** where selected bytes came from—workspace, module
  cache, vendor tree, local replacement, or `GOROOT`—plus applied overlays.
  Acquisition never substitutes for provenance or semantic identity.
- **implementation identity:** stable identity for one concrete emitted or
  manual implementation, including generic or representation specialization.
- **semantic model:** the minimum target-independent representation needed to
  state exact Go operations and effects. It is not emitted TypeScript.
- **fact:** a whole-program truth derived once from the semantic model.
- **plan:** an immutable, total decision selecting storage, calls, copies,
  dispatch, modules, and target operations.
- **generated unit:** a declaration or executable body carrying a valid
  GoToTS ownership marker and matching post-format baseline hash.
- **manual unit:** a declaration/body without a generated marker, or one whose
  current post-format body hash differs from its generated baseline.
- **placeholder:** a generated, typed, throwing body for an explicitly
  unsupported or externally owned implementation.
- **product root:** an explicitly selected executable, public API, test, or
  extension assembly root.
- **certified:** materialized, strict-typechecked, reachable, executed where
  applicable, independently verified, and covered by a passing publication
  attestation.

## WCBUBWHB Requirement

Every design, implementation, review, test, and generated-surface change begins
with WCBUBWHB analysis:

1. Show the observed source and artifact problem.
2. Search for the complete semantic class and sibling paths.
3. Name the one authoritative truth owner.
4. Fix the highest layer that can eliminate the class.
5. Delete the wrong abstraction, duplicate state, fallback, or alternate route.
6. State the simplest exact result without preserving current implementation
   shape.
7. Measure staticness, source size, typecheck cost, runtime, extension, and
   consumer consequences.
8. Show source -> semantic decision -> produced TypeScript examples.
9. Establish independent differential/contract and mutation proof.
10. Prove by broad search that no sibling or replaced path remains.

A second occurrence of the same workaround class stops feature work and
reopens the owning abstraction.

## Non-Negotiable Invariants

- One semantic fact has one authoritative producer and one typed identity.
- Every Go construct occurrence has exactly one contextual semantic
  disposition; unknown and unclassified are hard failures.
- Parents assign grammatical roles to child constructs; children do not infer
  meaning by inspecting parents or source text.
- Whole-program facts are sealed before planning; planning is total and
  immutable before lowering.
- Lowering performs no type checking, semantic discovery, representation
  choice, or corpus-specific recognition.
- One current implementation path exists. No fallback, compatibility reader,
  old/new mode, retry with changed semantics, or duplicate state survives.
- No semantic value is recovered through `any`, `unknown`, unchecked casts,
  reflection, spelling lookup, source-text scanning, or dynamic host-shape
  inspection.
- Definitions are owned once. References may repeat.
- Ordinary Go calls remain ordinary TypeScript calls; hidden semantic
  arguments require local typed necessity evidence and are never the default.
- Generated code never invokes through `Function.prototype.call`, `apply`, or
  `bind`, and never uses prototype lookup/manipulation to recover Go method
  semantics. Method adapters are ordinary statically typed functions or
  lambdas selected by a closed plan.
- Generated output is source-shaped and cost-bounded from the first construct,
  not optimized after semantic completion.
- Generated and manual declarations may coexist in one file; ownership is per
  declaration/body, never per file.
- Manual code requires only TypeScript code. No user-authored registry attaches
  it to generated output.
- Regeneration starts from an empty generated baseline and never preserves old
  generated text.
- Reachability traverses generated, manual, runtime, standard-library,
  external, extension, initialization, function-value, and dynamic-dispatch
  edges before publication or pruning.
- Customer extension behavior remains supported through one typed extension
  contract; extensions consume finalized evidence and never re-enter semantic
  analysis.
- Compilation and verification are deterministic and have no LLM dependency.

## Honest Support States

Each selected declaration, initializer, function literal, and body has exactly
one current state:

- `automatic`: completely generated by a proven lowering;
- `manual`: implemented by accepted manual TypeScript;
- `external`: implemented through a resolved external or standard-library
  contract;
- `placeholder`: typed but unresolved and therefore publication-blocking; or
- `unsupported`: analysis can identify the exact construct but no permitted
  materialization exists.

These states are separate from processing stages such as inventoried,
semantically analyzed, planned, emitted, typechecked, executed, and certified.
Reports must name both the support state and the highest completed stage.

## Reading Order

- [`architecture.md`](architecture.md) defines ownership, pipeline, dependency
  walls, project structure, and context-aware analysis.
- [`translation.md`](translation.md) defines the Go construct inventory,
  semantic operations, planning rules, and representative output.
- [`completion.md`](completion.md) defines runtime, standard library,
  externals, extensions, manual ownership, regeneration, and reachability.
- [`verification.md`](verification.md) defines independent proof, architecture
  gates, performance controls, and the eighteen publication gates.
- [`delivery.md`](delivery.md) defines the cleanroom implementation order,
  deletion policy, upgrade process, and final acceptance checklist.

## Completion

GoToTS may be described as complete for a declared Go version and target
profile only when:

- the language catalog is exhaustive for that Go version;
- every selected occurrence has one exact disposition;
- every reachable implementation is automatic, accepted manual, or resolved
  external;
- no reachable placeholder or unsupported construct remains;
- two unrelated Go projects and the first large acceptance corpus pass;
- regeneration preserves reachable manual work and identifies obsolete work;
- the existing extension contract passes its customer compatibility suite;
- generated architecture and costs pass their calibrated budgets; and
- all eighteen gates pass with zero blocked, failed, waived, or stale results
  at the exact clean published revision.
