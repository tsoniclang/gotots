# `@gotots/gostdlib`

`@gotots/gostdlib` provides Go standard-library behavior for TypeScript
produced by GoToTS.

Public ESM subpaths mirror Go import paths:

```ts
import * as strings from "@gotots/gostdlib/strings.js";
import * as os from "@gotots/gostdlib/os.js";
import * as filepath from "@gotots/gostdlib/path/filepath.js";
```

The selected implementation uses Node.js internally. Backend names, contract
digests, and compiler ABI names are not part of the public API.
