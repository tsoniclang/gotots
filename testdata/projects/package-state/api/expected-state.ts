import type { int32 } from "@gotots/runtime/scalars.js";
import { GoStorageFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export class $PackageState {
    declare Start: int32;
    declare private readonly then?: never;
}
export const $state = new $PackageState();
attribute<$PackageState>().add(GoStorageFact, "gotots-go-package-storage-fact-v1", "package-state", "example.com/package-state/api", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/api/state.ts", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", "$state", 1);
