# ADR 0001: Initial TypeScript-Go Source Pin

Date: 2026-07-15
Status: accepted
Owner: gotots-maintainers
Implementation revision: `58f4d8b`
Schema/ABI impact: `pin-schema-v3`
Spec reference: `docs/spec/01-input-census-publication.md` and
`docs/spec/13-governance-upgrades.md`

## Context

Translation evidence is meaningful only against one immutable source and
toolchain identity. A branch name, version string, or clean checkout does not
attest parser/type-checker behavior or selected source bytes.

## Alternatives

- Track an upstream branch: rejected because the input can change without a
  GoToTS revision.
- Pin source but accept any same-version Go toolchain: rejected because binary
  and GOROOT contents can differ.
- Select another initial revision: rejected because the accepted TSTS and
  translator evidence is tied to the recorded revision with no compensating
  product requirement.

## Decision

The first gotots target is `github.com/microsoft/typescript-go` at revision
`c78d39e7075b4fc641b12b1f35d905c54cdc13ef`, extracted with Go `go1.26.4
linux/amd64` under the `linux-amd64` build profile.

The pin is recorded in `pins/typescript-go.json` and verified fail-closed by
`internal/pinning` before any extraction: exact revision, clean checkout,
module identity, and complete toolchain identity — version, GOOS/GOARCH, the
go executable sha256, and a digest of the GOROOT VERSION file plus the
complete GOROOT/src tree. Two different toolchains reporting the same
version string do not pass. Checkout cleanliness is re-verified after
loading to prove extraction mutated nothing.

## Effects

Every census and generated bundle carries the source/toolchain identity.
Changing either creates a separate upgrade domain and requires regeneration;
there is no compatibility path between evidence from different pins.

## Migration

Inputs and reports that do not satisfy pin schema v3 are regenerated from the
committed pin. They are not upgraded heuristically or accepted by version
string alone.

## Evidence

Registry entry: `docs/decisions/registry.json#ADR-0001`. Proof artifacts:
`pins/typescript-go.json`, `internal/pinning`, and the attestation fixtures.

The committed pin, attestation fixtures, and TSTS reference revision provide
the cross-checkable evidence. Machine-local checkout count and active-shell
toolchain state are not durable proof.

## Reconsider when

- The product selects a newer TypeScript-Go release for TSTS.
- The upgrade-repeatability milestone needs a second pin; that pin
  is chosen fresh and does not modify this one.
