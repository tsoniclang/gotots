# ADR 0005: Inline Hash-Owned Manual Bodies

Date: 2026-07-17
Status: accepted
Owner: gotots-maintainers
Implementation revision: `eabaabb`
Schema/ABI impact: `manual-ownership-v1`
Spec reference: `docs/spec/08-externals-manual-extensions.md`,
`docs/spec/10-machine-contracts-diagnostics.md`, and
`docs/spec/13-governance-upgrades.md`
Registry entry: `docs/decisions/registry.json#ADR-0005`

## Context

GoToTS needs a practical completion path for rare bodies and external library
operations that automatic lowering does not yet implement. The prior design
required a developer to edit generated code, explicitly promote the body,
serialize a separate typed-AST artifact, and maintain a registration record.
That duplicated the implementation, dependency graph, and ownership truth and
made regeneration depend on high-ceremony metadata rather than the code being
maintained.

The product also needs all generated code to remain disposable during a source
upgrade, while preserving manually implemented bodies in the same file or class
as freshly regenerated code. Manual source must remain statically analyzable and
must not create a legacy fallback or runtime choice.

## Alternatives

- Separate manual source files plus a JSON attachment manifest: rejected
  because it duplicates identity, imports, dependencies, and reachability that
  the typed AST already provides.
- A promoted canonical body-AST artifact outside generated roots: rejected
  because developers would maintain code and a second ownership artifact, and
  generated/manual bodies could not coexist naturally in one class.
- Preserve any file containing an edit: rejected because unrelated old
  generated declarations, helpers, imports, and bodies would survive an
  upgrade.
- Text patches or marker-delimited statement ranges: rejected because they are
  syntax-fragile, cannot prove semantic identity, and create partial ownership.
- Always overwrite and require manual changes to be reapplied: rejected because
  regeneration would destroy intentional maintained source.

## Decision

One generated file carries a mandatory versioned header. Every independently
owned executable body carries an adjacent canonical implementation ID and the
SHA-256 of its exact body text after pinned deterministic formatting. The
immutable accepted generated baseline, not the editable marker alone, attests
that hash.

Generated and manual bodies may coexist in one file and class. A normalized
current body matching its attested baseline hash is generated and disposable;
a differing body or an absent marker is manual. A file without the verified
generated header is manual. Malformed, forged, duplicate, or ambiguous evidence
blocks. Nested bodies reconcile structurally deepest-first so one edit has one
owner.

The file header applies only to declarations proven by the attested baseline.
An unmarked developer-added helper function or class inside a generated file is
manual source, remains graph-discovered without registration metadata, and is
never discarded merely because its containing file is generated.

GoToTS emits exact typed throwing placeholders for unavailable owned bodies and
external operations. Replacing the throw with typed TypeScript is the entire
manual implementation workflow. No user-authored JSON, registration,
dependency list, promotion artifact, or runtime registry exists.

Regeneration extracts manual AST bodies from the attested old baseline and
editable workspace, creates a complete new generated baseline in an empty
staging root, discards all old generated units, structurally overlays valid
manual bodies, regenerates imports, and derives the complete typed dependency
and reachability graph. It reports current, placeholder, stale, missing,
unreachable, orphaned, automatically-lowerable, and invalid states.

Manual bodies remain the sole implementation until an explicit hash-guarded
reset selects the new generated body. Unreachable/orphaned manual source is
preserved by default and may be removed only by a separate dry-run/apply prune.

## Effects

- Manual completion requires editing TypeScript only.
- Per-body ownership allows generated and manual methods to coexist without
  preserving an obsolete generated file.
- Every source upgrade produces an exact machine delta and reachability result.
- External stubs use the same ownership mechanism as owned difficult bodies.
- Derived ledgers and acceptance evidence remain machine-readable, but are not
  user-maintained semantic inputs.
- A manual body receives no weaker semantic, staticness, test, or performance
  standard than generated code.
- There is one selected implementation and no compatibility or fallback path.

## Migration

The separate promotion/artifact workflow is removed rather than supported in
parallel. Implementation introduces `manual-ownership-v1`, generated headers,
per-body markers, immutable baseline attestation, structural reconciliation,
derived manual ledgers, reset, and prune as one contract. Existing generated
trees without an attested baseline are not guessed to be generated; they are
regenerated or treated as manual input and reconciled fail-closed.

The contract is accepted here before product support exists. Until every gate
and mutation test in the governing specification is implemented, manual bodies
remain an unimplemented translator capability and cannot establish selected
product completion.

## Evidence

Registry entry: `docs/decisions/registry.json#ADR-0005`. Governing proof
artifacts at the contract revision are
`docs/spec/08-externals-manual-extensions.md`,
`docs/spec/10-machine-contracts-diagnostics.md`,
`docs/spec/11-testing-acceptance.md`, and `internal/policy/spec_test.go`.

Implementation acceptance additionally requires the positive, mutation,
regeneration, source-upgrade, reachability, reset, prune, strict-typecheck,
staticness, Go-differential, extension, and performance evidence enumerated by
the specification. No current coverage percentage substitutes for those gates.

## Reconsider when

- A simpler mechanism preserves mixed-file manual bodies across a fresh
  generation without user metadata or generated-file retention.
- The pinned formatter cannot provide deterministic body ranges and a typed-AST
  ownership proof provides equal low-ceremony behavior.
- Measured reconciliation cost violates the accepted regeneration budget.
- A concrete source upgrade demonstrates that canonical structural joining
  cannot preserve a valid manual body without ambiguity.
