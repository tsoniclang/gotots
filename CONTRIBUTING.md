# Contributing to gotots

gotots mechanically translates the complete declared TS-Go source corpus
into statically analyzable TypeScript — closed-world completeness over
the current corpus, fail-closed detection of future TS-Go idioms. It is
not a universal Go compiler. The committed specification under
`docs/spec/` governs the work. Start with `mission-and-scope.md`, then read
`translation-and-output.md`, `performance-and-representation.md`, and
`testing-and-acceptance.md`. The local `.analysis/scope/` packet contains
review context and candidate designs; it is not normative.

## Hard rules

- Go language semantics and the declared owned corpus only; every
  imported package — including the Go standard library — is external and
  receives a deterministic declaration/stub obligation, never a
  translator-embedded implementation.
- Every source-bearing unit gets exactly one disposition: automatically
  translated, declared manual body, external stub obligation, or
  explicitly excluded. There is no implicit fifth category.
- No source-name guessing, regex semantic rewriting, reflection, dynamic
  dispatch, compatibility paths, or silent fallback. Unknown constructs,
  directives, or type variants fail closed before partial output is
  published — passing the current corpus never permits unknown constructs.
- Fix defects at the highest correct abstraction (WCBUBWHB): a problem
  repeated across sites indicates a missing generic lowering, not
  permission for repeated patches. Manual exceptions are explicit,
  enumerated, and stale-detected — never implicit.
- Evidence is identity-bearing and deterministic; aggregates derive from
  records. Never weaken a test or gate to make a result pass; green tests
  alone never establish coverage — dispositions do.

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
- Before every commit: `test -z "$(gofmt -l .)"`, `go vet ./...`,
  `go test -count=1 ./...`, and `git diff --check` must all pass; before
  every push additionally `go test -count=1 -race ./...`.
- Follow the ordered gates in `docs/spec/testing-and-acceptance.md`; a later
  product suite never substitutes for an earlier completeness proof.
- Add a source-linked Go input, typed decision, generated TypeScript shape,
  semantic oracle, and staticness/performance evidence for each semantic
  lowering class.
- Continue through the complete declared scope. Census, declaration,
  vertical-slice, focused-test, and corpus milestones are checkpoints rather
  than completion conditions.
