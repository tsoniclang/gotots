import type { int32 } from "../../../support/scalars.js";
import { $state } from "../../../packages/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/api/state.js";
import { $state as $state__dep } from "../../../packages/30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6/dep/package.js";
export function Run(): int32 {
    $state__dep.B++;
    $state__dep.Trace++;
    $state.Start++;
    return $state.Start * 10000 + $state__dep.A * 1000 + $state__dep.B * 100 + $state__dep.Trace;
}
