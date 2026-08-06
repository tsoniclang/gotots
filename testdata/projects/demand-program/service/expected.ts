import type { int32 } from "../../../runtime/scalars.js";
import { Even as Even__from_mathx } from "../../../packages/9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b/mathx/package.js";
export function Compute(value: int32): int32 {
    let Even = value;
    Even += Even__from_mathx(value);
    return Even;
}
