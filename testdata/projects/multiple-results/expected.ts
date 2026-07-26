import type { bool, int32 } from "../../../support/scalars.js";
export function Pair(value: int32): [
    int32,
    bool
] {
    return [(value + (1 as int32)) | 0, value >= (0 as int32)];
}
export function Forward(value: int32): [
    int32,
    bool
] {
    return Pair(value);
}
export function Consume(value: int32): int32 {
    const __gotots_results_0: [
        int32,
        bool
    ] = Pair(value);
    let next: int32 = __gotots_results_0[0];
    let positive: bool = __gotots_results_0[1];
    if (positive) {
        return next;
    }
    return value;
}
export function Reassign(value: int32): int32 {
    let next: int32 = value;
    let positive: bool = false as bool;
    const __gotots_results_1: [
        int32,
        bool
    ] = Pair(value);
    next = __gotots_results_1[0];
    positive = __gotots_results_1[1];
    if (positive) {
        return next;
    }
    return value;
}
export function KeepFirst(value: int32): int32 {
    const __gotots_results_2: [
        int32,
        bool
    ] = Pair(value);
    let next: int32 = __gotots_results_2[0];
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
    return [value, (value + (2 as int32)) | 0];
}
export function Add(left: int32, right: int32): int32 {
    return (left + right) | 0;
}
export function AddPair(value: int32): int32 {
    const __gotots_results_3: [
        int32,
        int32
    ] = Numbers(value);
    return Add(__gotots_results_3[0], __gotots_results_3[1]);
}
