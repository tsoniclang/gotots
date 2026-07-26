import type { int32 } from "../../../support/scalars.js";
import { Compute as Compute__from_example__u2e_com__u2f_demand__u2f_service } from "../service/service.js";
export const Compute: int32 = 5 as int32;
export function Run(value: int32): int32 {
    return (Compute__from_example__u2e_com__u2f_demand__u2f_service(value) + Compute) | 0;
}
