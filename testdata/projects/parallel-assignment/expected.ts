import type { int64 } from "@tsonic/core/types.js";
export function SwapLeft(left: int64, right: int64): int64 {
    const $assign0: int64 = right;
    const $assign1: int64 = left;
    left = $assign0;
    right = $assign1;
    return left;
}
export function Rotate(current: int64, next: int64): int64 {
    const $assign2: int64 = next;
    const $assign3: int64 = current;
    current = $assign2;
    let previous: int64 = $assign3;
    return previous;
}
export function Declare(left: int64, right: int64): int64 {
    const $assign4: int64 = left;
    const $assign5: int64 = right;
    let first: int64 = $assign4;
    let second: int64 = $assign5;
    return first + second;
}
export function Shadow(value: int64): int64 {
    if (true) {
        const $assign6: int64 = value + (1 as int64);
        const $assign7: int64 = value;
        let value$1: int64 = $assign6;
        let previous: int64 = $assign7;
        return value$1 + previous;
    }
    return 0 as int64;
}
