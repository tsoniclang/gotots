import type { int64 } from "@tsonic/core/types.js";
export function Sum(limit: int64): int64 {
    let total: int64 = 0 as int64;
    for (let current: int64 = 0 as int64; current < limit; current++) {
        if (current === 2 as int64) {
            continue;
        }
        total += current;
    }
    return total;
}
