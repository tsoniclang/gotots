import type { bool, int64 } from "@tsonic/core/types.js";
import { Pair as Pair__from_example__u2e_com__u2f_results__u2f_producer } from "../producer/producer.js";
export function Run(value: int64): int64 {
    const __gotots_results_0: [
        int64,
        bool
    ] = Pair__from_example__u2e_com__u2f_results__u2f_producer(value);
    let next: int64 = __gotots_results_0[0];
    let zero: bool = __gotots_results_0[1];
    if (zero) {
        return next;
    }
    return value;
}
