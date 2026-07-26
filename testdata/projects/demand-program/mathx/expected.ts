import type { int32 } from "../../../support/scalars.js";
export const Offset: int32 = 2 as int32;
export function Even(value: int32): int32 {
    if (value === 0 as int32) {
        return Offset;
    }
    return Odd((value - (1 as int32)) | 0);
}
export function Odd(value: int32): int32 {
    if (value === 0 as int32) {
        return 0 as int32;
    }
    return Even((value - (1 as int32)) | 0);
}
