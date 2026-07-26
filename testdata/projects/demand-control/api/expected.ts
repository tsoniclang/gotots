import type { int64 } from "@tsonic/core/types.js";
import { Sum as Sum__from_example__u2e_com__u2f_control__u2f_worker } from "../worker/worker.js";
export function Run(value: int64): int64 {
    switch (value) {
        case 0 as int64: {
            return 0 as int64;
            break;
        }
        default: {
            return Sum__from_example__u2e_com__u2f_control__u2f_worker(value);
            break;
        }
    }
}
