import type { int64 } from "@tsonic/core/types.js";
export function Classify(value: int64): int64 {
    let result: int64 = 0 as int64;
    {
        let current: int64 = value;
        switch (current) {
            case 0 as int64: {
                let branch: int64 = 10 as int64;
                result = branch;
                break;
            }
            case 1 as int64:
            case 2 as int64: {
                let branch: int64 = 20 as int64;
                result = branch;
                break;
            }
            default: {
                let branch: int64 = 30 as int64;
                result = branch;
                break;
            }
        }
    }
    return result;
}
