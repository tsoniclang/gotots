import type { bool, int64 } from "@tsonic/core/types.js";
export function Pair(value: int64): [
    int64,
    bool
] {
    return [value + (1 as int64), value === 0 as int64];
}
