import type { bool, int32 } from "@gotots/runtime/scalars.js";
export function Pair(value: int32): [
    int32,
    bool
] {
    return [value + 1, value >= 0];
}
export function Forward(value: int32): [
    int32,
    bool
] {
    return Pair(value);
}
export function Consume(value: int32): int32 {
    const __gotots_results_0 = Pair(value);
    let next = __gotots_results_0[0];
    let positive = __gotots_results_0[1];
    if (positive) {
        return next;
    }
    return value;
}
export function Reassign(value: int32): int32 {
    let next = value;
    let positive = false;
    const __gotots_results_1 = Pair(value);
    next = __gotots_results_1[0];
    positive = __gotots_results_1[1];
    if (positive) {
        return next;
    }
    return value;
}
export function KeepFirst(value: int32): int32 {
    const __gotots_results_2 = Pair(value);
    let next = __gotots_results_2[0];
    return next;
}
export function Discard(value: int32): int32 {
    Pair(value);
    return value;
}
export function Numbers(value: int32): [
    int32,
    int32
] {
    return [value, value + 2];
}
export function Add(left: int32, right: int32): int32 {
    return left + right;
}
export function AddPair(value: int32): int32 {
    const __gotots_results_3 = Numbers(value);
    return Add(__gotots_results_3[0], __gotots_results_3[1]);
}
