import type { int32 } from "../../../support/scalars.js";
export function SwapLeft(left: int32, right: int32): int32 {
    const __gotots_assign_0 = right;
    const __gotots_assign_1 = left;
    left = __gotots_assign_0;
    right = __gotots_assign_1;
    return left;
}
export function Rotate(current: int32, next: int32): int32 {
    const __gotots_assign_2 = next;
    const __gotots_assign_3 = current;
    current = __gotots_assign_2;
    let previous = __gotots_assign_3;
    return previous;
}
export function Declare(left: int32, right: int32): int32 {
    const __gotots_assign_4 = left;
    const __gotots_assign_5 = right;
    let first = __gotots_assign_4;
    let second = __gotots_assign_5;
    return first + second;
}
export function Shadow(value: int32): int32 {
    if (true) {
        const __gotots_assign_6 = value + 1;
        const __gotots_assign_7 = value;
        let value__shadow_1 = __gotots_assign_6;
        let previous = __gotots_assign_7;
        return value__shadow_1 + previous;
    }
    return 0;
}
export function Accumulate(total: int32, delta: int32): int32 {
    total += delta;
    return total;
}
