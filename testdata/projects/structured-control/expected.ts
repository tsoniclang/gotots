import type { int32 } from "../../../runtime/scalars.js";
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
        total = total + current;
        current++;
    }
    return total;
}
export function Once(): int32 {
    let total = 0;
    for (;;) {
        total = total + 1;
        break;
    }
    return total;
}
