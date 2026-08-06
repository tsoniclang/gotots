import type { int32 } from "../../../runtime/scalars.js";
export function Classify(value: int32): int32 {
    let result = 0;
    {
        let current = value;
        switch (current) {
            case 0: {
                let branch = 10;
                result = branch;
                break;
            }
            case 1:
            case 2: {
                let branch = 20;
                result = branch;
                break;
            }
            default: {
                let branch = 30;
                result = branch;
                break;
            }
        }
    }
    return result;
}
