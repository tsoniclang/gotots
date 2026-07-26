import type { bool, int64 } from "@tsonic/core/types.js";
export function SwapLeft(left: int64, right: int64): int64 {
    const __gotots_assign_0: int64 = right;
    const __gotots_assign_1: int64 = left;
    left = __gotots_assign_0;
    right = __gotots_assign_1;
    return left;
}
export function Rotate(current: int64, next: int64): int64 {
    const __gotots_assign_2: int64 = next;
    const __gotots_assign_3: int64 = current;
    current = __gotots_assign_2;
    let previous: int64 = __gotots_assign_3;
    return previous;
}
export function Declare(left: int64, right: int64): int64 {
    const __gotots_assign_4: int64 = left;
    const __gotots_assign_5: int64 = right;
    let first: int64 = __gotots_assign_4;
    let second: int64 = __gotots_assign_5;
    return first + second;
}
export function Shadow(value: int64): int64 {
    if (true as bool) {
        const __gotots_assign_6: int64 = value + (1 as int64);
        const __gotots_assign_7: int64 = value;
        let value__shadow_1: int64 = __gotots_assign_6;
        let previous: int64 = __gotots_assign_7;
        return value__shadow_1 + previous;
    }
    return 0 as int64;
}
export function Accumulate(total: int64, delta: int64): int64 {
    total += delta;
    return total;
}
