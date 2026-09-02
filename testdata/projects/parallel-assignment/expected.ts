import type { int32 } from "@gotots/runtime/scalars.js";
export function SwapLeft(left: int32, right: int32): int32 {
    const assignmentValue = right;
    const assignmentValue2 = left;
    left = assignmentValue;
    right = assignmentValue2;
    return left;
}
export function Rotate(current: int32, next: int32): int32 {
    const assignmentValue3 = next;
    const assignmentValue4 = current;
    current = assignmentValue3;
    let previous = assignmentValue4;
    return previous;
}
export function Declare(left: int32, right: int32): int32 {
    const assignmentValue5 = left;
    const assignmentValue6 = right;
    let first = assignmentValue5;
    let second = assignmentValue6;
    return first + second | 0;
}
export function Shadow(value: int32): int32 {
    if (true) {
        const assignmentValue7 = value + 1 | 0;
        const assignmentValue8 = value;
        let value__shadow_1 = assignmentValue7;
        let previous = assignmentValue8;
        return value__shadow_1 + previous | 0;
    }
    return 0;
}
export function Accumulate(total: int32, delta: int32): int32 {
    const assignmentValue9 = delta;
    total = total + assignmentValue9 | 0;
    return total;
}
