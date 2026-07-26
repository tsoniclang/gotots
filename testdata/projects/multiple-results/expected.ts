import type { bool, int64 } from "@tsonic/core/types.js";
export function Pair(value: int64): [
    int64,
    bool
] {
    return [value + (1 as int64), value >= (0 as int64)];
}
export function Forward(value: int64): [
    int64,
    bool
] {
    return Pair(value);
}
export function Consume(value: int64): int64 {
    const __gotots_results_0: [
        int64,
        bool
    ] = Pair(value);
    let next: int64 = __gotots_results_0[0];
    let positive: bool = __gotots_results_0[1];
    if (positive) {
        return next;
    }
    return value;
}
export function Reassign(value: int64): int64 {
    let next: int64 = value;
    let positive: bool = false as bool;
    const __gotots_results_1: [
        int64,
        bool
    ] = Pair(value);
    next = __gotots_results_1[0];
    positive = __gotots_results_1[1];
    if (positive) {
        return next;
    }
    return value;
}
export function KeepFirst(value: int64): int64 {
    const __gotots_results_2: [
        int64,
        bool
    ] = Pair(value);
    let next: int64 = __gotots_results_2[0];
    return next;
}
export function Discard(value: int64): int64 {
    Pair(value);
    return value;
}
export function Numbers(value: int64): [
    int64,
    int64
] {
    return [value, value + (2 as int64)];
}
export function Add(left: int64, right: int64): int64 {
    return left + right;
}
export function AddPair(value: int64): int64 {
    const __gotots_results_3: [
        int64,
        int64
    ] = Numbers(value);
    return Add(__gotots_results_3[0], __gotots_results_3[1]);
}
