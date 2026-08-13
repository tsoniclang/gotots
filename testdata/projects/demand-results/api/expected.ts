import type { int32 } from "@gotots/runtime/scalars.js";
import { Pair as Pair__from_producer } from "../../../../packages/example.com/results/producer/package.js";
export function Run(value: int32): int32 {
    const __gotots_results_0 = Pair__from_producer(value);
    let next = __gotots_results_0[0];
    let zero = __gotots_results_0[1];
    if (zero) {
        return next;
    }
    return value;
}
