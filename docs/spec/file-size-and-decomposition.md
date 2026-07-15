# Normative Policy: File Size And Semantic Decomposition

Status: normative. Enforced repository-wide by
`internal/policy` (`TestRepositoryFileSizes`). This document is part of the
committed gotots specification; it supplements the design scope packet.

## Policy

- No hand-maintained implementation or test source file may exceed
  **600 physical lines**.
- Files approaching the limit are decomposed proactively, before they
  become monoliths.
- Splits follow **semantic responsibilities** with meaningful filenames.
  Each resulting file has one coherent responsibility, clear ownership,
  and a narrow internal API.
- Forbidden split shapes: `part1`/`part2` fragments, `helpers2`, numbered
  shards, arbitrary line-based cuts, and thin forwarding files that
  preserve a hidden monolith.
- A split yields exactly one implementation path.
- The gate scans the entire repository. There is no allowlist, and no file
  is exempt.
- Reproducibly generated artifacts are handled by fixing their generator's
  output organization, not by normalizing oversized generated monoliths.

## Enforcement

`internal/policy/filesize.go` walks the repository, counts physical lines
deterministically, and reports every violation as:

```text
GOTOTS_FILE_TOO_LARGE:
internal/example/monolith.go has 742 lines; maximum is 600.
Split it by semantic responsibility.
```

`TestRepositoryFileSizes` fails on any violation;
`TestCheckTreeRejectsOversizedFile` proves the gate itself rejects a
violation. The gate excludes only trees that contain no hand-maintained
gotots source: VCS state, local scratch/evidence areas (`.analysis`,
`.temp`, `.tests`), dependency caches, and `testdata` fixture content.

## Decomposition guidance

Split by model and invariant, so the filename tells future maintainers
where each responsibility lives. The census implementation is the worked
example:

```text
internal/census/
  model.go        report data model
  run.go          pipeline orchestration and blockers
  load.go         typed loading and package-role evidence
  analyze.go      per-file analysis and import recording
  reconcile.go    pass-1/pass-2 file-set reconciliation
  identity.go     record identity uniqueness validation
  aggregate.go    deterministic ordering and derived aggregates
  inspect.go      typed AST/type inspection
  publish.go      transactional bundle publication
```

Test suites split by the contract being proven, not by count:

```text
internal/census/
  census_fixture_test.go       census invariants on a fixture repo
  attestation_fixture_test.go  injection/tamper/replacement rejection
  publication_test.go          publication safety contracts
  publication_overlap_test.go  output/source containment fixture
```
