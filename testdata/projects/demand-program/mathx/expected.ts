import type { int64 } from "@tsonic/core/types.js";
export const Offset: int64 = 2 as int64;
export function Even(value: int64): int64 {
    if (value === 0 as int64) {
        return Offset;
    }
    return Odd(value - (1 as int64));
}
export function Odd(value: int64): int64 {
    if (value === 0 as int64) {
        return 0 as int64;
    }
    return Even(value - (1 as int64));
}
