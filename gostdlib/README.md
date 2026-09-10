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

Provider value assignment preserves existing destination locations, including
record fields and array elements. It is distinct from allocating a copy.
The serial `sync.Map` and `sync.Pool` providers reject copying an instance
after first use, which their Go API contracts prohibit. Assigning a fresh
instance resets an existing destination without retargeting retained pointers.
This rejection is a provider boundary, not a claim that Go detects every
invalid copy at runtime.
