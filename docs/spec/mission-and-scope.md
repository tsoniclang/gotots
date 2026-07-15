# GoToTS Mission And Scope

Status: normative. This is the canonical committed statement of what
GoToTS is and is not; every other document must agree with it.
Machine-enforceable requirements have validation gates
(`internal/policy`); this document governs the rest.

## Mission

> Mechanically translate the complete declared TS-Go source corpus into
> high-quality, statically analyzable TypeScript, with exact semantic
> preservation, deterministic regeneration, and automatic detection of new
> constructs introduced by future TS-Go revisions.

Two requirements carry equal weight:

1. **Total corpus coverage.** Every source-bearing unit in the declared
   TS-Go port scope is accounted for. Every declaration, signature,
   constraint, body, and semantic operation in that scope receives an
   exact disposition. Nothing is silently ignored or approximated.
2. **Generic, scalable foundations.** Supported behavior is expressed in
   terms of Go syntax and semantics — never TS-Go filenames, function
   names, or one-off source patterns. When a future TS-Go revision
   introduces a new Go idiom, the census detects it and generation fails
   closed until the idiom is implemented correctly, through the highest
   reusable semantic abstraction. The architecture admits new idioms
   without redesigning the translator.

This is closed-world completeness over the current TS-Go corpus, with
open-ended detection and extension for future TS-Go revisions.

## Explicit Non-Goal

GoToTS is **not** a general-purpose Go-to-TypeScript compiler and does not
translate arbitrary Go programs unrelated to TS-Go.

A Go construct absent from the declared corpus does not need an
implementation — but it must be **recognized**. Example: if the corpus
contains no `select` statement, GoToTS implements no `select` lowering;
when a later TS-Go revision introduces one, the upgrade produces a
deterministic blocking result:

```text
GOTOTS_UNSUPPORTED_STATEMENT:
new select statement at checker/work.go:142
```

It must never silently omit the statement, approximate it, leave
untranslated Go syntax in output, continue with a warning, or special-case
the file that contains it. When implemented, the implementation models Go
`select` semantics as generally as the TS-Go usage requires.

## Declared Corpus Boundary

The translation scope is the complete in-scope TS-Go compiler
implementation, defined **mechanically** by the project profile
(`profiles/tsts/project.json`): canonical owned package roots, test-only
roots, categorized hard exclusions, tooling roots, and explicit build
profiles. Path shape never defines scope at analysis time.

- LSP implementation code is out of scope.
- Fourslash infrastructure is out of scope.
- Excluded code is still inventoried and carries a durable exclusion
  disposition; it never disappears from the universe.
- External libraries are represented through deterministic
  declaration/stub obligations.
- Test sources and test data are classified under the agreed test-porting
  policy, never silently mixed with product code.

## Total Disposition Requirement

Every source-bearing unit receives exactly one disposition:

1. automatically translated;
2. declared manual-body implementation;
3. external declaration/stub obligation;
4. explicitly excluded by the canonical scope.

There is no fifth, implicit category. Within automatically translated
code, every semantic operation is accounted for: if a required lowering
(for example, reslice semantics for `values = values[:count]`) is missing,
authoritative generation fails — it never emits something plausible and
relies on tests to discover the mistake.

## Genericity Versus Source-Specific Patching

A supported operation is implemented for the semantic class everywhere it
occurs: `values = values[:count]` becomes a semantic reslice operation
with correct aliasing, capacity, bounds, and nil behavior at every
occurrence; `field := &value.scope` exercises general address-of-field
semantics.

Repository paths, declaration names, and source spellings may appear in
diagnostics and provenance. They must never determine language semantics.
A conditional on a function or file name inside translation logic is a
defect regardless of whether it produces correct output.

## External Library Boundary

GoToTS translates the Go **language**; it implements no library. This
applies to third-party modules and to library-oriented standard packages
alike (`strings.Builder`, `bytes.Buffer`, `io.Reader`, `sync.Mutex`).
A use such as `builder.WriteString(text)` produces a deterministic
external declaration/stub obligation preserving the referenced package,
type, member, signature, and call semantics; the separate emulation layer
implements it later.

Language-level structures and operations remain translator
responsibilities: indexing, map access and comma-ok, `append`, reslicing,
`len`/`cap`, channel operations, goroutine semantics, and every other
predeclared operation — classified by `go/types` identity, never by
spelling.

## Performance And Representation

Generated TypeScript should be idiomatic and efficient, but optimization
is evidence-based and semantics-preserving. Representation selection is
**static selection from one semantic model**; there is exactly one
implementation path:

```text
Go slice semantics
        |
        +-- proven simple usage  --> JavaScript array
        |
        +-- richer observed usage --> explicit slice carrier
```

A direct representation requires proof (for a slice: nil/empty
indistinguishable, capacity unobserved, no aliased reslices, no dependence
on append reallocation). Absent proof, the semantics-preserving
representation is used. Every choice is recorded and testable; none is
selected by source name or optimistic fallback.

## Future TS-Go Upgrade Contract

Every new TS-Go pin follows this process:

1. Attest the exact upstream revision and toolchain.
2. Rebuild the complete source census.
3. Compare the new and previous source universes.
4. Compare declarations, signatures, type parameters, constraints,
   constants, body operations, directives, external dependencies, and
   semantic idioms.
5. Report every added, removed, or changed requirement.
6. Block generation on every unrecognized construct or changed invariant.
7. Add support through the highest reusable semantic abstraction.
8. Add focused semantic tests and corpus regressions.
9. Regenerate from clean inputs.
10. Run determinism, differential, self-compilation, common-project, and
    performance gates.

The correct response to a new idiom (for example a first integer-range
loop) is to implement the semantic class, never to patch the first
occurrence.

## Manual-Body Contract

Manual implementations are permitted only through an explicit,
deterministic contract. Each manual body records:

- canonical Go declaration identity;
- the reason automatic translation is not currently suitable;
- the manually owned TypeScript implementation;
- signature compatibility proof;
- regeneration behavior;
- tests proving the boundary;
- whether the exception remains necessary after each upgrade.

Manual code lives outside generated files; regeneration never overwrites
it, and generated output never depends on hand-edited markers inside
generated files. The report enumerates every exception; unknown or stale
exceptions fail validation. Manual exceptions are not an unbounded escape
hatch — a repeated exception pattern indicates a missing generic lowering.

## Repository Organization

The file-size and semantic-decomposition policy in
`docs/spec/file-size-and-decomposition.md` is part of this specification
and enforced repository-wide by `internal/policy`.

## Forbidden Framings

No GoToTS document, report, or implementation comment may state or imply:

- that GoToTS is a universal Go compiler;
- that passing the current corpus permits unknown constructs;
- that source-name patches are acceptable because the input is TS-Go;
- that external libraries should be implemented by the translator;
- that only current TS-Go idioms need to be recognized;
- that manual body overrides may be implicit;
- that tests alone establish complete coverage;
- that oversized files may remain as debt.
