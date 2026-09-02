import type { int32 } from "@gotots/runtime/scalars.js";
export function Classify(value: int32): int32 {
    {
        let current = value;
        if (current < 0) {
            return -1;
        }
        else if (current === 0) {
            return 0;
        }
        else {
            return 1;
        }
    }
}
export function Sum(limit: int32): int32 {
    let total = 0;
    let current = 0;
    for (; current < limit;) {
        total = total + current | 0;
        current = current + 1 | 0;
    }
    return total;
}
export function Once(): int32 {
    let total = 0;
    for (;;) {
        total = total + 1 | 0;
        break;
    }
    return total;
}
