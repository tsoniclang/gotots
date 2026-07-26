import type { int64 } from "@tsonic/core/types.js";
import { Compute as Compute__from_example__u2e_com__u2f_demand__u2f_service } from "../service/service.js";
export const Compute: int64 = 5 as int64;
export function Run(value: int64): int64 {
    return Compute__from_example__u2e_com__u2f_demand__u2f_service(value) + Compute;
}
