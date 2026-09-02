import type { bool, int32 } from "@gotots/runtime/scalars.js";
import { Base, Enabled } from "./constants.js";
export function AddBase(value: int32): int32 {
    return Base + value | 0;
}
export function IsEnabled(): bool {
    return Enabled;
}
