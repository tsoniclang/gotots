import type { int32 } from "../../../support/scalars.js";
export function Sum(limit: int32): int32 {
    let total: int32 = 0 as int32;
    for (let current: int32 = total; current < limit; current = (current + 1) | 0) {
        if (current === 2 as int32) {
            continue;
        }
        total = (total + current) | 0;
    }
    return total;
}
