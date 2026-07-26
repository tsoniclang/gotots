import type { int32 } from "../../../support/scalars.js";
import { Pair as Pair__from_producer } from "../../../packages/cefd26c72d61128807e2cdeb08e5aeba9b304e837e50b30474e46b45e203ec8e/producer/package.js";
export function Run(value: int32): int32 {
    const __gotots_results_0 = Pair__from_producer(value);
    let next = __gotots_results_0[0];
    let zero = __gotots_results_0[1];
    if (zero) {
        return next;
    }
    return value;
}
