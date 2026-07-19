# Input, Census And Publication

## Closed Input Contract

Every run identifies:

- source repository and immutable revision;
- selected project profile and content hash;
- Go executable version and digest;
- GOROOT source digest;
- GOOS, GOARCH, build tags, experiments, and the complete environment
  allowlist;
- TypeScript compiler, JavaScript runtime, module resolver, strict
  configuration, and generated-helper/runtime revisions and digests;
- GoToTS implementation revision;
- external contract, extension seam, and emulation inputs;
- the current editable mixed-source workspace and its immutable prior
  generated-baseline attestation for manual-body reconciliation;
- schema and decision-registry versions; and
- output and proof-format versions.

The selected initial profile is `linux-amd64` with the exact toolchain in the
pin. Additional profiles are independent acceptance domains.

## Attest Before Execute

The Go executable digest and compiled-in frontend version are verified before
the executable performs source discovery. The environment is normalized:

- `GOTOOLCHAIN=local`;
- `GOWORK=off`;
- read-only module resolution;
- no network;
- no user workspace, replacement, or cache authority outside recorded inputs;
- explicit `GOENV`, `GOFLAGS`, module/vendor, overlay, cache, cgo, locale,
  timezone, and process-option values; and
- rejection or clearing of every unlisted environment input before child
  execution.

Every selected source byte is reconciled against the pinned Git blob. A clean
working tree alone is insufficient because ignored and injected files can still
affect package loading.

## Scope Filter Before Census

The profile establishes selected roots and completely outside roots before the
frontend enumerates packages. LSP, fourslash, and editor-service roots are
outside. No record for a file below those roots enters the source, declaration,
operation, external, or test ledgers.

After filtering, GoToTS computes the selected package dependency closure.
Closure into an outside root fails with `GOTOTS_SCOPE_DEPENDENCY_OUTSIDE`.

## Selected Census

The selected census records:

- packages and canonical package identities;
- source and generated files actually selected by the Go toolchain;
- declarations, methods, signatures, constraints, aliases, constants, and
  variables;
- bodies, initializers, statements, expressions, and semantic operations;
- build directives and compiler intrinsics;
- generic declarations and instantiations;
- imported external declarations and transitive type closure;
- test entities selected by the test policy; and
- every support and ownership state.

Counts derive from identity-bearing records. A summary count without records is
not evidence.

## Canonical Identity

Identity comes from typed package and object evidence, including test-package
identity where selected. Blank identifiers may repeat only where Go permits and
are qualified by declaration position. Generated IDs are stable functions of
canonical source identity, never traversal order or emitted spelling.

The census rejects duplicate IDs, conflicting ownership, overlapping selected
and outside roots, path escape, malformed spans, unresolved selected imports,
and source bytes not backed by the pin.

## Operation Census

Operations are classified by semantic shape rather than call site text. At
minimum each record carries:

- operation kind;
- operand and result semantic types;
- selected declaration or builtin identity;
- nilability, addressability, and comparability;
- mutation, alias, escape, and copy effects;
- generic instantiation;
- source span;
- enclosing implementation identity; and
- initial support classification.

Large repeated classes such as append, reslice, map access, string indexing,
address-taking, assertions, and generic instantiation have individual machine
records. Implementation decisions are reviewed once per semantic class.

## Support And Ownership Accounting

Every selected declaration has one declaration owner. Every selected body or
initializer has one implementation owner and one support state. Accepted
combinations are:

| Declaration owner | Implementation state | Meaning |
| --- | --- | --- |
| generated-core | generated | automatic IR/lowering ownership; evidence stage reported separately |
| generated-core | accepted-manual | hash-detected complete structural body that passed applicable gates |
| generated-core | unimplemented | recognized implementation withheld |
| external-contract | no-source-body | typed external obligation |
| extension-core-contract | separate-extension | generated typed seam |

No ignored, miscellaneous, inferred-manual, partially generated, or warning-only
state exists.

## Denominator Reconciliation

Every report defines its denominator and derivation. Production bodies,
translation candidates, synthetic initializers, declaration-only objects,
generated toolchain units, and selected tests are separate classes.

For selected function and method bodies, every run reports identity-bearing
counts for each translation evidence stage from `authority-scope.md`. It
also reports separately:

- selected packages;
- packages containing directly unimplemented units;
- packages withheld transitively through dependency closure;
- retained runnable packages;
- selected declarations that are not bodies;
- unsupported non-body declarations;
- selected function literals and synthetic initializers; and
- retained runnable body artifacts.

The report includes exact joins from each earlier stage to the next. A body
that is IR-admitted and lowered but belongs to a withheld package remains in
those denominators and is absent from `module-retained`. It cannot be counted
as emitted, runnable, typechecked, executed, or certified.

Operation-site counts, operation-bearing-body counts, and deduplicated unions
are distinct. Overlapping semantic classes never have their body counts added
without an identity join. A percentage always names its numerator and
denominator; “coverage”, “translated”, and “completed” are invalid denominator
labels.

If two reports present different body totals, a machine join must account for
every difference by canonical identity and disposition. Historical totals are
never embedded as normative constants.

## Deterministic Reports

Canonical JSON uses versioned schemas, sorted identity keys, normalized paths,
and deterministic numeric/string encoding. Reports contain no timestamps,
machine-local roots, process IDs, or nondeterministic traversal order in their
hashed content.

The required bundle includes:

- input attestation;
- selected census;
- declaration and implementation ownership;
- operation classes and site membership;
- external obligations;
- support coverage;
- per-body translation evidence stages and their gate/artifact references;
- representation plans;
- unimplemented diagnostics;
- generated-baseline/body-hash evidence and the derived manual status/delta
  ledger;
- body-materialization artifacts for every automatically lowered body;
- generated artifact inventory;
- test results; and
- manifest hashes.

## Staged Generation

Automatic translation always generates a complete new baseline under an empty
staging root. It does not incrementally modify, copy, or consult old generated
bodies for semantic decisions. Before generation, a separate reconciliation
phase may read the attested prior baseline and current editable mixed-source
workspace solely to identify manual bodies by the contract in
`externals-manual-extensions.md`. After the new baseline exists, every valid
manual AST unit participates in one complete candidate graph. Only units reached
by the fixed-point traversal are overlaid; imports and final graph evidence are
then regenerated from the assembled typed AST. Unknown graph edges block apply
rather than preserving or removing source heuristically.

Generation writes only beneath a newly created staging root whose real path is
contained by the configured output parent. Paths are canonical-encoded and
checked against traversal, symlink escape, case collision, reserved names, and
duplicate ownership.

Generated files and manifests are flushed before publication. Publication is an
atomic rename of the complete staged tree. A failed or blocked run leaves the
accepted product tree untouched.

## Unimplemented Publication Rule

An unimplemented unit may appear in analysis and coverage reports and as an
exact typed throwing placeholder in the editable workspace. That placeholder
is a completion surface, not a runnable implementation. Any package or entry
artifact whose dependency closure reaches it is withheld and named in the
diagnostic. Independent closed packages may be emitted into an explicitly
partial, non-product staging bundle whose manifest declares `complete: false`.

A partial bundle contains reports and IR plus the editable placeholder tree,
declaration-only output, or runnable modules whose complete transitive
implementation closure is closed. A package containing an unresolved
placeholder or omitted body is never represented as a retained runnable
module, and no import path may make that implementation appear available.

Every automatically lowered body, including one in a withheld package, retains
an analysis-only canonical TypeScript body AST or equivalent canonical body
serialization and its hash. This artifact exists for audit and regeneration;
it is not a module, cannot be imported or executed, and cannot satisfy
`module-retained`. Keeping it does not create a second implementation path.

A proof may name an emitted module artifact only when that artifact exists and
contains the identified body. A body in a withheld module instead records its
body-materialization artifact, withholding reason, and affected closure. A
`generatedFile`-style field pointing to an absent file is invalid evidence.

Product publication requires `complete: true` and zero reachable
unimplemented units.

## Reproducibility

Two runs from identical attested inputs into empty staging roots produce
byte-identical generated source, source maps, contracts, ownership records,
proofs, diagnostics, and manifests. The verifier reconstructs every bundle hash
without trusting generator summaries.
