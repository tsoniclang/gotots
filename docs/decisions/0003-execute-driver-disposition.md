# ADR 0003: internal/execute Driver Disposition

Date: 2026-07-15
Status: accepted
Spec reference: `docs/spec/mission-and-scope.md` "Declared Corpus Boundary",
"Total Disposition Requirement"

## Decision

The upstream tsc command-line driver family — the root package
`internal/execute` (`tsc.go`, `watcher.go`), its `--build` driver mode
`internal/execute/build` (whose tests run through the driver harness),
and the harness `internal/execute/tsctests` — is **unselected**: outside
the mechanically translated corpus, inventoried with a durable
disposition. The clean subpackages `internal/execute/incremental` and
`internal/execute/tsc` remain owned.

## Evidence

At pin `c78d39e7`, the contamination is confined to exactly one file:
`internal/execute/tsc.go` imports the hard-excluded editor-service
packages `internal/format` and `internal/ls/lsutil` (formatter and
language-service output paths of the CLI). The subpackages have no editor-service
dependencies and no dependency on the root driver package; `tsctests`
imports the driver directly, `execute/build`'s tests import `tsctests`,
and the only owned-side consumer is the test-support
`testutil/harnessutil` via `execute/incremental`, which stays owned.

TSTS `main` (`a80a5da3`) ships a **manually trimmed** port of the driver
(`src/internal/execute/tsc.ts`) that omits the editor-service paths. That
trimming is product behavior with no exact upstream body — precisely the
class the specification assigns to product-owned assembly, not mechanical
translation.

## Consequences

- The census carries zero contradiction edges at the pin; census bundles
  become authoritative (subject to clean generator provenance).
- The TSTS CLI surface is product-owned assembly. When the translator
  reaches CLI integration, the driver either gains a reviewed
  seam/manual-body disposition or remains product source; either path is
  an explicit future decision, not a silent default.

## Reconsider when

- Upstream decouples the driver from editor-service packages (the
  contradiction the census would then no longer report), or
- the product decides to generate the CLI surface, which requires a
  reviewed disposition for the formatter/lsutil bindings.
