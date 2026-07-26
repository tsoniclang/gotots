import type { int32 } from "../../../support/scalars.js";
export function Classify(value: int32): int32 {
    let result: int32 = 0 as int32;
    {
        let current: int32 = value;
        switch (current) {
            case 0 as int32: {
                let branch: int32 = 10 as int32;
                result = branch;
                break;
            }
            case 1 as int32:
            case 2 as int32: {
                let branch: int32 = 20 as int32;
                result = branch;
                break;
            }
            default: {
                let branch: int32 = 30 as int32;
                result = branch;
                break;
            }
        }
    }
    return result;
}
