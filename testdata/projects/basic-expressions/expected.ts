import type { bool, int32 } from "../../../support/scalars.js";
export function Arithmetic(value: int32): int32 {
    return Math.imul(((value - (3 as int32)) | 0), 2 as int32);
}
export function WrapAdd(value: int32): int32 {
    return (value + (1 as int32)) | 0;
}
export function WrapSubtract(value: int32): int32 {
    return (value - (1 as int32)) | 0;
}
export function WrapMultiply(value: int32): int32 {
    return Math.imul(value, 2 as int32);
}
export function Increment(value: int32): int32 {
    value = (value + 1) | 0;
    return value;
}
export function Decrement(value: int32): int32 {
    value = (value - 1) | 0;
    return value;
}
export function Compare(left: int32, right: int32): [
    bool,
    bool,
    bool,
    bool,
    bool,
    bool
] {
    return [left === right, left !== right, left < right, left <= right, left > right, left >= right];
}
export function Logic(left: bool, right: bool): bool {
    return (left && !right) || (!left && right);
}
export function Never(): bool {
    for (;;) {
    }
}
export function ShortCircuitAnd(): bool {
    return false as bool && Never();
}
export function ShortCircuitOr(): bool {
    return true as bool || Never();
}
