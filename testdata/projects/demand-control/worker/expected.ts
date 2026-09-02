import type { int32 } from "@gotots/runtime/scalars.js";
export function Sum(limit: int32): int32 {
    let total = 0;
    for (let current = total; current < limit; current = current + 1 | 0) {
        if (current === 2) {
            continue;
        }
        const assignmentValue = current;
        total = total + assignmentValue | 0;
    }
    return total;
}
