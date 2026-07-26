import type { bool, int32 } from "../../../support/scalars.js";
export function SwapLeft(left: int32, right: int32): int32 {
    const __gotots_assign_0: int32 = right;
    const __gotots_assign_1: int32 = left;
    left = __gotots_assign_0;
    right = __gotots_assign_1;
    return left;
}
export function Rotate(current: int32, next: int32): int32 {
    const __gotots_assign_2: int32 = next;
    const __gotots_assign_3: int32 = current;
    current = __gotots_assign_2;
    let previous: int32 = __gotots_assign_3;
    return previous;
}
export function Declare(left: int32, right: int32): int32 {
    const __gotots_assign_4: int32 = left;
    const __gotots_assign_5: int32 = right;
    let first: int32 = __gotots_assign_4;
    let second: int32 = __gotots_assign_5;
    return (first + second) | 0;
}
export function Shadow(value: int32): int32 {
    if (true as bool) {
        const __gotots_assign_6: int32 = (value + (1 as int32)) | 0;
        const __gotots_assign_7: int32 = value;
        let value__shadow_1: int32 = __gotots_assign_6;
        let previous: int32 = __gotots_assign_7;
        return (value__shadow_1 + previous) | 0;
    }
    return 0 as int32;
}
export function Accumulate(total: int32, delta: int32): int32 {
    total = (total + delta) | 0;
    return total;
}
