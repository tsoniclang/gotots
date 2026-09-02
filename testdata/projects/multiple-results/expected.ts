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
    const results = Pair(value);
    let next = results[0];
    let positive = results[1];
    if (positive) {
        return next;
    }
    return value;
}
export function Reassign(value: int32): int32 {
    let next = value;
    let positive = false;
    const results2 = Pair(value);
    next = results2[0];
    positive = results2[1];
    if (positive) {
        return next;
    }
    return value;
}
export function KeepFirst(value: int32): int32 {
    const results3 = Pair(value);
    let next = results3[0];
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
    const results4 = Numbers(value);
    return Add(results4[0], results4[1]);
}
