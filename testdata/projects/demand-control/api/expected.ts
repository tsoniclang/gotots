import type { int32 } from "../../../support/scalars.js";
import { Sum as Sum__from_example__u2e_com__u2f_control__u2f_worker } from "../worker/worker.js";
export function Run(value: int32): int32 {
    switch (value) {
        case 0 as int32: {
            return 0 as int32;
            break;
        }
        default: {
            return Sum__from_example__u2e_com__u2f_control__u2f_worker(value);
            break;
        }
    }
}
