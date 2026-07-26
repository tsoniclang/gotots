import type { int64 } from "@tsonic/core/types.js";
export function Classify(value: int64): int64 {
    {
        let current: int64 = value;
        if (current < (0 as int64)) {
            return (0 as int64) - (1 as int64);
        }
        else if (current === 0 as int64) {
            return 0 as int64;
        }
        else {
            return 1 as int64;
        }
    }
}
export function Sum(limit: int64): int64 {
    let total: int64 = 0 as int64;
    let current: int64 = 0 as int64;
    for (; current < limit;) {
        total = total + current;
        current++;
    }
    return total;
}
export function Once(): int64 {
    let total: int64 = 0 as int64;
    for (;;) {
        total = total + (1 as int64);
        break;
    }
    return total;
}
