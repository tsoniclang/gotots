# Checker Resolution Only

`npm run core:resolve` installs the compiler's existing resolution-only core
fixture in the provider's ignored `node_modules/`. That fixture owns all marker
declarations used by canonical runtime source, including typed pointers. The
old partial declaration copy in this directory is deleted. Provider and runtime
checking resolve the same declarations through ordinary ESM package resolution.

The fixture is not a runtime implementation, a marker selector, or an output
dependency. Its pointer/layout operation bodies throw if reached; they never
implement executable location semantics. Its `struct`/`field` declaration
fixtures provide only the existing checker scaffold, not a physical value
representation or execution proof. Ordinary provider tests exercise the native
logical string/slice paths. Pointer/layout execution requires the selected
target integration proofs, not these standalone tests. Provider assembly copies
only production source; canonical product checking resolves core through the
shared owner and finalizes its exact facts.

The provider certificate exact-joins unsafe-pointer callable positions to this
installed import's declaration identity. A private same-shaped provider declaration does
not satisfy that join. Executable pointer values must come from generated
canonical operations, never from this declaration or an assertion.
