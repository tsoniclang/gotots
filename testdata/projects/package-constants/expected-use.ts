import type { bool, int64 } from "@tsonic/core/types.js";
import { Base, Enabled } from "./constants.js";
export function AddBase(value: int64): int64 {
    return Base + value;
}
export function IsEnabled(): bool {
    return Enabled;
}
