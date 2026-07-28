import type { int32 } from "../../../support/scalars.js";
import { $state } from "../../../packages/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/dep/state.js";
export type Cell$Storage = {
    Value: int32;
};
export class Cell {
    declare private readonly $goType: void;
    private constructor(private readonly $storage: Cell$Storage) {
    }
    public static $make($field0: int32): Cell {
        return new Cell({
            Value: $field0
        });
    }
    public static $storageOf($source: Cell): Cell$Storage {
        return $source.$storage;
    }
    public static $fromStorage($source: Cell$Storage): Cell {
        return new Cell($source);
    }
    public get Value(): int32 {
        return this.$storage.Value;
    }
    public set Value($value: int32) {
        this.$storage.Value = $value;
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
    return $state.A * 10000 + $state.B * 1000 + $state.Trace * 10 + $state.hidden + $state.__go___proto__ + Cell.$fromStorage($state.Empty).Value + Cell.$fromStorage($state.Filled).Value;
}
