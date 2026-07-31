import type { bool, int32 } from "../../../runtime/scalars.js";
import { Base, Enabled } from "./constants.js";
export function AddBase(value: int32): int32 {
    return Base + value;
}
export function IsEnabled(): bool {
    return Enabled;
}
