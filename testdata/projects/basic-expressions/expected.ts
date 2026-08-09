import type { bool, int32 } from "@gotots/runtime/scalars.js";
export function Arithmetic(value: int32): int32 {
    return (value - 3) * 2;
}
export function WrapAdd(value: int32): int32 {
    return value + 1;
}
export function WrapSubtract(value: int32): int32 {
    return value - 1;
}
export function WrapMultiply(value: int32): int32 {
    return value * 2;
}
export function Increment(value: int32): int32 {
    value++;
    return value;
}
export function Decrement(value: int32): int32 {
    value--;
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
    return false && Never();
}
export function ShortCircuitOr(): bool {
    return true || Never();
}
