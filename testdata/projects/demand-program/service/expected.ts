import type { int32 } from "@gotots/runtime/scalars.js";
import { Even as Even__from_mathx } from "../../../../packages/example.com/demand/mathx/package.js";
export function Compute(value: int32): int32 {
    let Even = value;
    Even += Even__from_mathx(value);
    return Even;
}
