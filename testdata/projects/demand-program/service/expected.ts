import type { int32 } from "../../../support/scalars.js";
import { Even as Even__from_example__u2e_com__u2f_demand__u2f_mathx } from "../mathx/math.js";
export function Compute(value: int32): int32 {
    let Even: int32 = value;
    Even = (Even + Even__from_example__u2e_com__u2f_demand__u2f_mathx(value)) | 0;
    return Even;
}
