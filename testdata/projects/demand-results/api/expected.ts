import type { int32 } from "@gotots/runtime/scalars.js";
import { Pair as Pair__from_producer } from "../../../../packages/example.com/results/producer/package.js";
export function Run(value: int32): int32 {
    const results = Pair__from_producer(value);
    let next = results[0];
    let zero = results[1];
    if (zero) {
        return next;
    }
    return value;
}
