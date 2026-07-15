# Contributing to gotots

gotots is a project-focused Go-to-TypeScript compiler whose first product
is a regenerated TSTS compiler. The design scope packet (`.analysis/scope/`,
local) and the committed specification under `docs/spec/` govern the work.

## Hard rules

- Go language semantics and selected owned project source only; every
  imported package — including the Go standard library — is external.
- No source-name guessing, regex semantic rewriting, reflection, dynamic
  dispatch, compatibility paths, or silent fallback. Unknown constructs
  fail closed before partial output is published.
- Fix defects at the highest correct abstraction (WCBUBWHB): a problem
  repeated across sites indicates a missing abstraction, not permission
  for repeated patches.
- Evidence is identity-bearing and deterministic; aggregates derive from
  records. Never weaken a test or gate to make a result pass.

## File size and decomposition (normative)

No hand-maintained implementation or test source file may exceed
**600 physical lines**. Decompose proactively, by semantic responsibility,
with meaningful filenames — never `part1`/`helpers2` shards or forwarding
facades, and never keep the original monolith as a compatibility path.
This is enforced repository-wide by `internal/policy`
(`TestRepositoryFileSizes`); see
`docs/spec/file-size-and-decomposition.md`.

## Workflow

- All accepted edits land serially on the single active feature branch.
- Commit coherent checkpoints after each completed architectural
  invariant, and keep the remote up to date.
- `gofmt`, `go vet ./...`, and `go test ./...` must be green before every
  commit; `go test -race` before every push.
