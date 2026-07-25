# GoToTS

GoToTS is a general Go-to-TypeScript compiler.

The project is being rebuilt around one direct architecture:

```text
Go AST + go/types
        |
        v
context-aware construct handlers
        |
        v
exact TS-Go AST
        |
        v
strict ESM TypeScript
```

There is no custom semantic IR, planning IR, lowering IR, or text emitter.
The governing contract is [`docs/spec/`](docs/spec/README.md).

Implementation advances capability by capability from focused failing
construct tests. Each accepted case builds typed TS-Go AST, passes strict
TypeScript checks, and proves behavior before neighboring cases are admitted.

Superseded implementations remain available on the pushed `archive/*`
branches. They are historical evidence and test-idea donors, not production
dependencies.
