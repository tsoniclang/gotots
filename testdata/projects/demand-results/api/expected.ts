import type { int32 } from "../../../support/scalars.js";
import { Pair as Pair__from_example__u2e_com__u2f_results__u2f_producer } from "../producer/producer.js";
export function Run(value: int32): int32 {
    const __gotots_results_0 = Pair__from_example__u2e_com__u2f_results__u2f_producer(value);
    let next = __gotots_results_0[0];
    let zero = __gotots_results_0[1];
    if (zero) {
        return next;
    }
    return value;
}
