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

This branch is intentionally a clean baseline containing authority and project
metadata only. Implementation is added capability by capability after the
relevant design and verification contract is accepted.

Superseded implementations remain available on the pushed `archive/*`
branches. They are historical evidence and test-idea donors, not production
dependencies.
