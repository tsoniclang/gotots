# GoToTS Specification

## Authority

These five files are the complete governing specification:

1. `README.md` — product boundary and vocabulary;
2. `architecture.md` — direct compiler architecture and project structure;
3. `translation.md` — contextual construct translation contract;
4. `verification.md` — proof and regression requirements; and
5. `delivery.md` — implementation order and completion criteria.

Historical branches, plans, reports, and implementations are evidence only.
They cannot override this specification. Requirements use **must** and
**must not**.

## Mission

Given a valid selected Go project, toolchain, build configuration, and explicit
environment contracts, GoToTS produces deterministic, readable, strict ESM
TypeScript with equivalent observable behavior.

It is a general Go compiler. `typescript-go` is an acceptance corpus, not a
production dependency, language authority, package-name exception, or source
of privileged translation rules.

Generated TypeScript must remain inside the statically compilable Tsonic
subset. Translating GoToTS itself and compiling that output through Tsonic is a
final product proof.

## One Compilation Model

```text
selected files and package graph
        |
        v
standard Go parser + one coherent go/types graph
        |
        v
one parent-directed, context-aware emission walk
        |
        v
generated Go bindings for the pinned TS-Go AST schema
        |
        v
one TS-Go formatter
        |
        v
strict TypeScript modules
```

The source program is represented only by the selected Go AST and its
`go/types` evidence. The target program is represented only by TS-Go AST nodes.
There is no compiler-defined source inventory artifact, semantic IR, operation
graph, whole-program plan, lowering IR, handwritten target tree, or target-text
fallback between them.

This does not prohibit ordinary compiler coordination. Deterministic names,
scope builders, imports, target declarations, diagnostics, references into the
Go graph, and placement requests are allowed. They must not restate the source
program as a second model or become a second semantic truth owner.

## Vocabulary

- **Go construct:** a grammar form represented by Go syntax, such as an
  assignment, call, declaration, index, receive, return, or function literal.
- **construct occurrence:** one concrete Go AST node in one selected file.
- **context:** facts supplied by the parent handler, such as expression versus
  statement position, assignment target, expected arity/type, lexical scope,
  and evaluation boundary.
- **semantic evidence:** the selected toolchain's `go/types` facts for the
  existing Go AST, including object bindings, types, constants, instances,
  selections, and signatures.
- **handler:** the one contextual translation owner for a construct family.
- **emission result:** TS-Go AST nodes plus explicit placement requests and
  diagnostics. It is target output under construction, not an IR.
- **placement request:** a typed request to insert an import, declaration,
  helper, temporary, or statement at a legal preferred target scope.
- **representation rule:** the direct rule choosing an exact TypeScript shape
  for a Go type, method, interface, value, or operation.
- **manual obligation:** an exact generated declaration whose implementation
  must be supplied manually.
- **true external:** unavailable or host/native behavior represented by an
  explicit contract rather than inferred from import spelling.

## Product Boundaries

All imported packages obey the same Go language. Toolchain metadata separately
identifies:

- workspace and source-available dependency packages, which are translated;
- standard-library declarations, whose selected-`GOROOT` contracts are
  generated and whose behavior is completed in reusable `gostdlib`;
- toolchain pseudo-packages and intrinsics, which have explicit compiler
  ownership; and
- true external, native, platform, `cgo`, or unsupported boundaries, which
  receive exact contracts and explicit placeholders.

No import-path prefix decides these classes.

## Non-Negotiable Results

- Every encountered construct is handled or fails with a typed unsupported
  diagnostic carrying source identity and context.
- Context-sensitive meanings are decided by the parent handler plus
  `go/types`, never by text or guesses.
- Every emitted TypeScript construct exists first as an exact TS-Go AST node.
- Definitions are owned once and output growth remains proportional to source
  complexity, not interface implementer count or package count.
- Generated code uses no `any`/`unknown` recovery, dynamic semantic lookup,
  `.call`, `.apply`, `.bind`, dynamic import, prototype patch, or source-text
  patching.
- Correct behavior, strict static typing, maintainable source shape, generated
  size, typecheck cost, generation cost, and runtime cost are simultaneous
  acceptance dimensions.
