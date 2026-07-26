import type { bool, int32 } from "../../../support/scalars.js";
export function Pair(value: int32): [
    int32,
    bool
] {
    return [(value + (1 as int32)) | 0, value === 0 as int32];
}
