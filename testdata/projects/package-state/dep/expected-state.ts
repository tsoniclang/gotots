import type { Cell$Storage as Cell__from_dep$Storage } from "../../../../modules/example.com/package-state/dep/state.js";
import type { int32 } from "@gotots/runtime/scalars.js";
export class $PackageState {
    A: int32;
    B: int32;
    Dormant: int32;
    Empty: Cell__from_dep$Storage;
    Filled: Cell__from_dep$Storage;
    Trace: int32;
    __go___proto__: int32;
    hidden: int32;
    constructor($initial0: int32, $initial1: int32, $initial2: int32, $initial3: Cell__from_dep$Storage, $initial4: Cell__from_dep$Storage, $initial5: int32, $initial6: int32, $initial7: int32) {
        this.A = $initial0;
        this.B = $initial1;
        this.Dormant = $initial2;
        this.Empty = $initial3;
        this.Filled = $initial4;
        this.Trace = $initial5;
        this.__go___proto__ = $initial6;
        this.hidden = $initial7;
    }
    declare private readonly then?: never;
}
export let $state: $PackageState;
export function $initializeState($initial0: int32, $initial1: int32, $initial2: int32, $initial3: Cell__from_dep$Storage, $initial4: Cell__from_dep$Storage, $initial5: int32, $initial6: int32, $initial7: int32): void {
    $state = new $PackageState($initial0, $initial1, $initial2, $initial3, $initial4, $initial5, $initial6, $initial7);
}
