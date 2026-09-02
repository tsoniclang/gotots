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
    static $zeroStorage(): Cell$Storage {
        return {
            Value: 0
        };
    }
    declare private readonly then?: never;
}
export function mark(value: int32): int32 {
    $state.Trace = globalThis.Math.imul($state.Trace, 10) + value | 0;
    $state.__go___proto__ = $state.__go___proto__ + 1 | 0;
    return value;
}
export function Snapshot(): int32 {
    return (((((globalThis.Math.imul($state.A, 10000) + globalThis.Math.imul($state.B, 1000) | 0) + globalThis.Math.imul($state.Trace, 10) | 0) + $state.hidden | 0) + $state.__go___proto__ | 0) + Cell.$storageOf(Cell.$fromStorage($state.Empty)).Value | 0) + Cell.$storageOf(Cell.$fromStorage($state.Filled)).Value | 0;
}
