import type { int32 } from "../../../runtime/scalars.js";
import { Compute as Compute__from_service } from "../../../packages/9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b/service/package.js";
export const Compute: int32 = 5;
export function Run(value: int32): int32 {
    return Compute__from_service(value) + Compute;
}
