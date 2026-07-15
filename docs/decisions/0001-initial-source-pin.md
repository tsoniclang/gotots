# ADR 0001: Initial TypeScript-Go Source Pin

Date: 2026-07-15
Status: accepted
Packet reference: `.analysis/scope/24-open-decisions.md` D1

## Decision

The first gotots target is `github.com/microsoft/typescript-go` at revision
`c78d39e7075b4fc641b12b1f35d905c54cdc13ef`, extracted with Go `go1.26.4
linux/amd64` under the `linux-amd64` build profile.

The pin is recorded in `pins/typescript-go.json` and verified fail-closed by
`internal/pinning` before any extraction: exact revision, clean checkout,
module identity, and toolchain identity (version, GOOS/GOARCH, executable
digest).

## Evidence

The candidate revisions in the packet (current TSTS `main` pin, the strongest
experimental Porter pin, a new latest pin) collapse to one choice, verified on
2026-07-15:

- TSTS `main` (`a80a5da3`) vendors `typescript-go` at `c78d39e7`.
- The experimental carrier-redesign worktree pins the same `c78d39e7`.
- Two clean local checkouts of that revision exist and verify clean.
- The pinned toolchain `go1.26.4 linux/amd64` is the active local toolchain.
- All existing Porter declaration evidence, semantic oracles, and TSTS test
  baselines were produced against this revision.

Choosing any other revision would discard the only existing cross-checkable
evidence while providing no product benefit at this phase.

## Reconsider when

- The product selects a newer TypeScript-Go release for TSTS.
- The upgrade-proof milestone (packet phase 11) needs a second pin; that pin
  is chosen fresh and does not modify this one.
