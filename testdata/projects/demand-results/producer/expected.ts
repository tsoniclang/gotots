import type { bool, int32 } from "@gotots/runtime/scalars.js";
export function Pair(value: int32): [
    int32,
    bool
] {
    return [value + 1 | 0, value === 0];
}
