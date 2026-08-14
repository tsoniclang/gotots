import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-state/dep/state.js";
export type Cell$Storage = {
    Value: int32;
};
export class Cell {
    declare private readonly $goType: void;
    public constructor(private readonly $storage: Cell$Storage) {
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
        return new Cell({
            Value: 0
        });
    }
    declare private readonly then?: never;
}
export function mark(value: int32): int32 {
    $state.Trace = $state.Trace * 10 + value;
    $state.__go___proto__++;
    return value;
}
export function Snapshot(): int32 {
    return $state.A * 10000 + $state.B * 1000 + $state.Trace * 10 + $state.hidden + $state.__go___proto__ + Cell.$storageOf(Cell.$fromStorage($state.Empty)).Value + Cell.$storageOf(Cell.$fromStorage($state.Filled)).Value;
}
