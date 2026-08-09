import type { int32 } from "@gotots/runtime/scalars.js";
import { Sum as Sum__from_worker } from "../../../packages/0cee61dcb0061ae1c4a32d0170cd19fe8dfb1fd69896ac0c0a272e22872549ae/worker/package.js";
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
