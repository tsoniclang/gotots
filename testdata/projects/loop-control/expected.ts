import type { int32 } from "../../../support/scalars.js";
export function Sum(limit: int32): int32 {
    let total = 0;
    for (let current = total; current < limit; current++) {
        if (current === 2) {
            continue;
        }
        total = total + current;
        if (total > 10) {
            break;
        }
    }
    return total;
}
