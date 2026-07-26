import type { int32 } from "../../../support/scalars.js";
export function Classify(value: int32): int32 {
    {
        let current: int32 = value;
        if (current < (0 as int32)) {
            return (0 as int32) - (1 as int32);
        }
        else if (current === 0 as int32) {
            return 0 as int32;
        }
        else {
            return 1 as int32;
        }
    }
}
export function Sum(limit: int32): int32 {
    let total: int32 = 0 as int32;
    let current: int32 = 0 as int32;
    for (; current < limit;) {
        total = (total + current) | 0;
        current = (current + 1) | 0;
    }
    return total;
}
export function Once(): int32 {
    let total: int32 = 0 as int32;
    for (;;) {
        total = (total + (1 as int32)) | 0;
        break;
    }
    return total;
}
