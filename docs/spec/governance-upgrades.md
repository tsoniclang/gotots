# Governance And Upgrades

## Decision Discipline

Architecture decisions choose concrete implementations within this
specification. Every accepted decision records:

- semantic problem and selected corpus evidence;
- alternatives;
- chosen mechanism;
- rejected simpler mechanisms;
- deterministic/static/performance effects;
- schemas and implementation revision;
- proof artifacts;
- migration;
- owner; and
- reopening conditions.

A decision cannot waive pinned Go semantics, complete accounting, fail-closed
behavior, one emitted implementation, or required gates.

## Closed Governing Decisions

The following govern implementation:

- selected TypeScript-Go corpus rather than arbitrary Go;
- LSP/fourslash/editor roots completely outside the input universe;
- typed Go frontend as semantic authority;
- complete semantic IR before lowering;
- machine classification by semantic class;
- monotonic fixed-point representation planning;
- simplest exact emitted TypeScript;
- class-level necessity before custom runtime machinery;
- static external contracts with separately owned emulation;
- one complete structural manual-body mechanism;
- declarative typed product-extension seams;
- explicit unimplemented support state;
- deterministic staged publication; and
- no LLM dependency in compilation or verification.

## Implementation Freedom

Private identifier spelling, internal data-structure organization, and
equivalent local lowering may evolve without an architecture decision when all
schemas, observable semantics, deterministic output, and performance gates
remain satisfied.

Changes to semantic class keys, representation lattices, ownership, public
machine schemas, custom mechanisms, manual workflow, extension seams, or
accepted implementation-permitted behavior require a decision update.

## Source Upgrade

For each pinned TypeScript-Go revision:

1. attest source, toolchain, prior immutable generated baseline, and current
   editable mixed-source workspace;
2. classify and extract manual AST units from verified post-format body hashes;
3. apply the scope filter;
4. rebuild selected package/declaration/body/operation/test census;
5. diff canonical identities and pre-planning operation-class membership;
6. detect added syntax, types, operations, directives, externals, and seams;
7. run fixed-point planning;
8. diff post-planning representation-class membership and plans;
9. classify every addition as implemented, manual candidate, external, or
   unimplemented;
10. regenerate the complete baseline into an empty staging root without
    copying old generated source;
11. structurally join valid manual candidates by exact current identity and
    signature without rewriting their bodies;
12. build the complete generated/manual/external/extension/test candidate graph
    and traverse it from the complete current roots to a deterministic fixed
    point;
13. overlay only reached manual units, regenerate imports, and verify the final
    graph;
14. report current, placeholder, stale, missing, unreachable, orphaned,
    reachability-unknown, automatically-lowerable, invalid, and planned-removal
    manual deltas;
15. run required gates; and
16. publish only the declared completion level when apply inputs still match
    the reviewed dry run.

Added uses of an implemented class require no design review when machine proofs
pass. A genuinely different class receives one class-level decision rather than
site patches.

Regeneration preserves manual bodies by canonical typed ownership, never by
retaining old generated files. Proven unreachable and orphaned manual units are
reported and omitted from the new active tree by the one regeneration
dry-run/apply path; they remain recoverable from the prior immutable bundle,
report, and version control. An incomplete graph blocks apply rather than
retaining or deleting heuristically. A newly automatically lowerable manual
body remains the sole implementation until an explicit identity-targeted reset
replaces it.

## Diffusion Workflow

Implementation expands by semantic class:

1. select the highest-volume or highest-value unimplemented class;
2. minimize representative source shapes;
3. add typed IR and constraints generically;
4. choose the simplest representation;
5. add differential/property tests;
6. let machine classification include all matching sites;
7. verify unrelated regions remain unchanged; and
8. update exact coverage.

There is no requirement to solve all difficult classes before useful
translation reports or independent partial artifacts exist. There is a strict
requirement to label every unsolved class and withhold affected product output.

## Highest-Abstraction Review

Before accepting a fix, reviewers ask:

- Is this selected from typed semantic evidence?
- Does it apply to the complete semantic class?
- Is an apparently local failure evidence of a missing IR operation,
  constraint, boundary effect, or verifier rule?
- Is ordinary TypeScript sufficient?
- Can typed whole-product static analysis eliminate the proposed runtime
  dispatch, lookup, carrier, or adapter?
- Is custom runtime behavior actually observed?
- Can a rare difficult body remain manual or unimplemented instead?
- Does the change introduce more than one implementation path?

Negative answers block the change.

## Repository Structure

Hand-maintained implementation, tests, and normative Markdown are at most 600
physical lines per file. Files split by semantic responsibility into meaningful
directories and names. Numeric shards, forwarding facades, and retained
monoliths are not decomposition.

The enforced limit is 600 physical lines per maintained file.

Generated artifacts are split at semantic declaration/dependency boundaries.
One declaration is not semantically rewritten only to satisfy a source-review
line limit.

Generated product trees live under manifest-owned staging/publication roots,
outside the maintained GoToTS source tree. If generated output is deliberately
checked into a maintained root, it becomes subject to that root's policy rather
than gaining a comment-based size exemption.

## Active Branch And Checkpoints

Work proceeds on one active feature branch. Coherent architectural and
implementation milestones are committed and pushed regularly. Remote branches
and tags are never deleted or force-pushed.

Meaningful work is not hidden in stashes. Scratch and failure evidence stay
under `.temp/`; local analysis stays under `.analysis/`; neither is
committed.

## Review Evidence

Every review message and pull request includes:

- exact branch/head;
- scope and completion level;
- changed semantic classes;
- generated/manual/unimplemented counts;
- manual current/placeholder/stale/missing/unreachable/orphaned/
  reachability-unknown/automatically-lowerable/invalid and removal counts;
- custom necessity decisions;
- gates run and exact results;
- deterministic report paths;
- performance comparison where relevant; and
- unresolved blockers.

Examples are representative, not limits on the required broad search.

## Specification Changes

Specification edits update `manifest.json`, cross-references, policy tests,
and affected decisions in the same change. A conflicting implementation does
not justify weakening the specification.

Normative language states the current contract. Historical exploration and
rejected drafts belong under uncommitted analysis, not the governing directory.

## Completion Checklist

Specification completion requires:

- one authoritative specification directory;
- exact manifest/index agreement;
- no unresolved contradiction;
- explicit support for honest unimplemented classes;
- complete semantic-family and boundary rules;
- fixed-point and custom-necessity contracts;
- external/manual/extension ownership;
- machine schemas and diagnostics;
- testing and performance gates;
- valid decision references;
- repository policy tests; and
- independent review.

Selected-product completion additionally requires:

- zero reachable unimplemented bodies;
- zero unresolved reachable external stubs;
- zero stale manual bodies;
- zero reachable manual placeholders, missing bodies, or invalid manual joins;
- zero reachability-unknown records and no proven unreachable/orphaned manual
  source in the active regenerated tree;
- every required extension seam assembled;
- complete selected tests and compiler corpus;
- deterministic regeneration;
- accepted performance; and
- atomic complete publication.

## Reopening

A decision is reopened by concrete counterexample, selected source change,
failed semantic oracle, verifier insufficiency, unacceptable measured
performance, or a demonstrably simpler exact mechanism. Convenience and
coverage percentage alone are insufficient.
