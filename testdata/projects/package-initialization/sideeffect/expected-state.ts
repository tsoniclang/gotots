import type { int32 } from "@gotots/runtime/scalars.js";
export class $PackageState {
    first: int32;
    second: int32;
    constructor($initial0: int32, $initial1: int32) {
        this.first = $initial0;
        this.second = $initial1;
    }
    declare private readonly then?: never;
}
export let $state: $PackageState;
export function $initializeState($initial0: int32, $initial1: int32): void {
    $state = new $PackageState($initial0, $initial1);
}
