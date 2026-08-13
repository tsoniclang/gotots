import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-state/api/state.js";
import { $state as $state__dep } from "../../../../packages/example.com/package-state/dep/package.js";
export function Run(): int32 {
    $state__dep.B++;
    $state__dep.Trace++;
    $state.Start++;
    return $state.Start * 10000 + $state__dep.A * 1000 + $state__dep.B * 100 + $state__dep.Trace;
}
