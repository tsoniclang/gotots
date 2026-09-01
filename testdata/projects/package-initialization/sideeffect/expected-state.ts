import type { int32 } from "@gotots/runtime/scalars.js";
import { GoStorageFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export class $PackageState {
    declare first: int32;
    declare second: int32;
    declare private readonly then?: never;
}
export const $state = new $PackageState();
attribute<$PackageState>().add(GoStorageFact, "gotots-go-package-storage-fact-v1", "package-state", "example.com/package-initialization/sideeffect", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/sideeffect/state.ts", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", "$state", 2);
