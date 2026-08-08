import type { int32 } from "../../../runtime/scalars.js";
import { $state } from "../../../packages/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/dep/state.js";
export class Cell {
    declare private readonly $goType: void;
    private constructor(public Value: int32) {
    }
    public static $make($field0: int32): Cell {
        return new Cell($field0);
    }
    static $zero(): Cell {
        return Cell.$make(0);
    }
}
export function mark(value: int32): int32 {
    $state.Trace = $state.Trace * 10 + value;
    $state.__go___proto__++;
    return value;
}
export function Snapshot(): int32 {
    return $state.A * 10000 + $state.B * 1000 + $state.Trace * 10 + $state.hidden + $state.__go___proto__ + $state.Empty.Value + $state.Filled.Value;
}
