# GoToTS Governing Specification

This directory is the sole normative product specification for GoToTS. GoToTS
mechanically translates the selected TypeScript-Go compiler corpus into
deterministic, statically analyzable TypeScript. It is project-focused rather
than a general Go translator.

The pinned Go source and selected build profile define semantic truth. Generated
TypeScript is accepted only when its observable behavior is permitted by that
Go program. Existing TypeScript, host behavior, convenient output, and
historical translator behavior are evidence rather than authority.

## Reading Order

1. `00-authority-scope.md`
2. `01-input-census-publication.md`
3. `02-compiler-semantic-ir.md`
4. `03-declarations-bodies-control.md`
5. `04-types-values-pointers.md`
6. `05-collections-strings.md`
7. `06-interfaces-generics-functions.md`
8. `07-packages-concurrency.md`
9. `08-externals-manual-extensions.md`
10. `09-representation-output.md`
11. `10-machine-contracts-diagnostics.md`
12. `11-testing-acceptance.md`
13. `12-performance.md`
14. `13-governance-upgrades.md`

All documents are normative. A narrower example cannot weaken a general rule.
Definitions in an earlier document govern later documents unless the later
document explicitly specializes the definition without weakening it.

## Central Contract

GoToTS performs the following deterministic pipeline:

1. attest the source, toolchain, profile, and product inputs;
2. build the exact selected source and declaration census;
3. create typed semantic IR for every selected declaration and body;
4. derive operation, effect, alias, escape, and boundary constraints;
5. propagate those constraints to a fixed point;
6. choose the simplest statically proven representation for every value-flow
   region;
7. lower to a typed TypeScript AST without semantic rediscovery;
8. reconcile the fresh generated baseline with automatically detected manual
   bodies, external implementations, and extension-owned inputs through one
   complete fixed-point candidate graph, overlaying only reached units;
9. validate the complete staged artifact; and
10. publish atomically.

The semantic model is complete even when output is simple. Information needed
only for regeneration, validation, or diagnostics remains in typed IR and
proof manifests. Runtime values carry only distinctions that selected code can
observe.

## Scope Boundary

LSP, fourslash, and editor-service implementation and tests are completely
outside the GoToTS input universe. Their files are not enumerated, parsed,
typed, counted, translated, stubbed, or entered into test ledgers. The profile
defines their excluded package roots before census. A dependency from selected
code into one of those roots is a profile error.

All imported packages outside the selected owned package set, including the Go
standard library, are external contracts. GoToTS translates language semantics
and emits typed external obligations; separately owned emulation code supplies
library behavior.

## Honest Incompleteness

A recognized semantic class or body may be classified `unimplemented`.
That classification is preferred to speculative lowering or an unjustified
runtime abstraction. It records exact identity, source span, typed operation,
reason, affected dependency closure, and required proof.

An unimplemented unit:

- is reported in machine coverage;
- may have an exact typed throwing placeholder only in the explicitly
  incomplete editable workspace;
- emits no placeholder that is counted as runnable or admitted to a product
  artifact;
- prevents publication of every dependent product artifact;
- does not prevent analysis or translation of independent units; and
- cannot be counted as translated, manually implemented, externally bound, or
  accepted product coverage.

Translator development may therefore advance by semantic class without
misrepresenting completeness. Complete selected-product acceptance requires no
reachable unimplemented unit.

## Progress Language

Progress reports distinguish support state from translation evidence. A body
with complete IR and automatic lowering is `generated`, but it is not thereby
present in a retained module, typechecked, executed, Go-equivalent, or
certified. Reports use the evidence stages defined in the authority and
scope chapter and publish the denominator for every count.

For example, “9,000 bodies are IR-admitted” says nothing by itself about how
many bodies are present in runnable retained modules. Package withholding can
remove thousands of otherwise lowered bodies from runnable output. No report
may shorten `ir-admitted` to “completed” or present its percentage as product
coverage.

## One Implementation

Each emitted artifact contains one statically selected implementation for every
operation. There is no runtime choice between representations, generated versus
manual bodies, exact versus fast helpers, or installed versus fallback
dependencies. Unknown evidence blocks before publication.

## Specification Inventory

`manifest.json` is the machine-readable inventory and reading order. Policy
tests require exact agreement among the manifest, this index, the filesystem,
cross-references, and the decision registry.
