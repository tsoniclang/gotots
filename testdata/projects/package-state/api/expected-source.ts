import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-state/api/state.js";
import { $state as $state__dep } from "../../../../packages/example.com/package-state/dep/package.js";
export function Run(): int32 {
    $state__dep.B = $state__dep.B + 1 | 0;
    $state__dep.Trace = $state__dep.Trace + 1 | 0;
    $state.Start = $state.Start + 1 | 0;
    return ((globalThis.Math.imul($state.Start, 10000) + globalThis.Math.imul($state__dep.A, 1000) | 0) + globalThis.Math.imul($state__dep.B, 100) | 0) + $state__dep.Trace | 0;
}
