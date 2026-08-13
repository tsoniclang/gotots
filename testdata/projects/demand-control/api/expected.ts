import type { int32 } from "@gotots/runtime/scalars.js";
import { Sum as Sum__from_worker } from "../../../../packages/example.com/control/worker/package.js";
export function Run(value: int32): int32 {
    switch (value) {
        case 0: {
            return 0;
            break;
        }
        default: {
            return Sum__from_worker(value);
            break;
        }
    }
}
