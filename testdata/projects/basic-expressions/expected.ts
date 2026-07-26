import type { bool, int64 } from "@tsonic/core/types.js";
export function Arithmetic(value: int64): int64 {
    return (value - (3 as int64)) * (2 as int64);
}
export function WrapAdd(value: int64): int64 {
    return value + (1 as int64);
}
export function WrapSubtract(value: int64): int64 {
    return value - (1 as int64);
}
export function WrapMultiply(value: int64): int64 {
    return value * (2 as int64);
}
export function IntWrapAdd(value: int64): int64 {
    return value + (1 as int64);
}
export function IntWrapSubtract(value: int64): int64 {
    return value - (1 as int64);
}
export function IntWrapMultiply(value: int64): int64 {
    return value * (2 as int64);
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
