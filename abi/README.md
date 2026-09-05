# Selected Go source ABI

This source provider owns the four `DataLayout` tokens imported by canonical
GoToTS output: `little32`, `little64`, `big32`, and `big64`, from the virtual
module `@gotots/abi/layout.js`. GoToTS selects the token from its explicit Go
build profile. Shared source-core validates the provider identity and writes
the resulting neutral layout facts. Targets do not interpret Go metadata.

Install this package locally; no publication is required. The TypeScript
capability entry is `@gotots/abi/typescript` and its capability identity is
`gotots.source-abi`. Select it through the host's existing capability selection.
Other target owners can compose `createGoAbiCapability(theirTargetId)` or the
target-neutral `goAbiCompilerContributions()` without changing this provider.
Do not register a second core extension: the host incorporates `dataLayouts`
into its one core extension.

```ts
import { little64 } from "@gotots/abi/layout.js";
import type { uint32 } from "@tsonic/core/types.js";
import { memoryLayout } from "@tsonic/core/lang.js";

const word = memoryLayout<uint32>(little64, 4, 4, 4);
```

The virtual module has no JavaScript implementation. A target must consume the
certified token and layout or reject the program before emission. There is no
ambient host detection, runtime registration, raw address registry, or target
memory implementation in this package.
