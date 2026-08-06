import type { int32 } from "../../../runtime/scalars.js";
export const Offset: int32 = 2;
export function Even(value: int32): int32 {
    if (value === 0) {
        return Offset;
    }
    return Odd(value - 1);
}
export function Odd(value: int32): int32 {
    if (value === 0) {
        return 0;
    }
    return Even(value - 1);
}
