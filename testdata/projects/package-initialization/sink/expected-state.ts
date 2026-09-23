import type { int32 } from "@gotots/runtime/scalars.js";
export class $PackageState {
    Count: int32;
    constructor($initial0: int32) {
        this.Count = $initial0;
    }
    declare private readonly then?: never;
}
export let $state: $PackageState;
export function $initializeState($initial0: int32): void {
    $state = new $PackageState($initial0);
}
