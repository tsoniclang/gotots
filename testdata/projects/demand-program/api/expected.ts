import type { int32 } from "@gotots/runtime/scalars.js";
import { Compute as Compute__from_service } from "../../../../packages/example.com/demand/service/package.js";
export const Compute: int32 = 5;
export function Run(value: int32): int32 {
    return Compute__from_service(value) + Compute;
}
