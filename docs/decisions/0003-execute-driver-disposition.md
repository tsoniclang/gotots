# ADR 0003: internal/execute Driver Disposition

Date: 2026-07-15
Status: accepted
Owner: gotots-maintainers
Implementation revision: `e56ed2e`
Schema/ABI impact: `profile-schema-v1`
Spec reference: `docs/spec/00-authority-scope.md` and
`docs/spec/01-input-census-publication.md`

## Context

The upstream command driver crosses into editor-service implementation that is
outside the selected product. Including only convenient files from that
dependency closure would violate the package/profile contract.

## Alternatives

- Include the complete driver and editor dependencies: rejected by the closed
  product scope.
- Trim individual functions during translation: rejected as source-specific
  partial ownership.
- Stub editor imports: rejected because it would make outside-universe source
  appear to be a permitted external library.

## Decision

The upstream tsc command-line driver family — the root package
`internal/execute` (`tsc.go`, `watcher.go`), its `--build` driver mode
`internal/execute/build` (whose tests run through the driver harness),
and the harness `internal/execute/tsctests` — is outside the selected input
universe through an explicit profile root. It is filtered before census. The
clean subpackages `internal/execute/incremental` and
`internal/execute/tsc` remain owned.

## Evidence

Registry entry: `docs/decisions/registry.json#ADR-0003`. Proof artifacts:
`profiles/tsts/project.json`, the census/profile fixtures, and revision
`e56ed2e`.

At pin `c78d39e7`, the contamination is confined to exactly one file:
`internal/execute/tsc.go` imports the outside-universe editor-service
packages `internal/format` and `internal/ls/lsutil` (formatter and
language-service output paths of the CLI). The subpackages have no editor-service
dependencies and no dependency on the root driver package; `tsctests`
imports the driver directly, `execute/build`'s tests import `tsctests`,
and the only owned-side consumer is the test-support
`testutil/harnessutil` via `execute/incremental`, which stays owned.

TSTS `main` (`a80a5da3`) ships a **manually trimmed** port of the driver
(`src/internal/execute/tsc.ts`) that omits the editor-service paths. That is
historical product evidence, not semantic authority. Every future pin must
justify the source disposition from the committed scope profile and typed
dependency graph.

## Effects

- The selected dependency closure carries no edge into the outside driver or
  editor-service roots.
- The TSTS CLI surface is product-owned assembly. When the translator
  reaches CLI integration, the driver either gains a reviewed
  seam/manual-body disposition or remains product source; either path is
  an explicit future decision, not a silent default.

## Migration

The selected profile excludes the declared driver roots before census and
retains only the independently closed subpackages. Existing manually trimmed
product code is historical evidence; generated ownership is not inferred from
it.

## Reconsider when

- Upstream decouples the driver from editor-service packages (the
  contradiction the census would then no longer report), or
- the product decides to generate the CLI surface, which requires a
  reviewed disposition for the formatter/lsutil bindings.
