import type { bool, int32 } from "../../../support/scalars.js";
export function Pair(value: int32): [
    int32,
    bool
] {
    return [value + 1, value === 0];
}
