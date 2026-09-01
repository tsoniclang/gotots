import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-state/api/state.js";
import { $state as $state__dep } from "../../../../packages/example.com/package-state/dep/package.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function Run(): int32 {
    $state__dep.B++;
    $state__dep.Trace++;
    $state.Start++;
    return $state.Start * 10000 + $state__dep.A * 1000 + $state__dep.B * 100 + $state__dep.Trace;
}
attribute<typeof Run>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-state/api|kind=4|receiver=|name=Run", "function", "example.com/package-state/api", "Run", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/package-state/api", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "modules/example.com/package-state/api/api.ts", "checked-syntax:api.go", "04bc19cfcb5a31e5715d223e550c0d48217238cdd5289c10f04cddd5452b5b39", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 87, 195, "go1.26", "");
attribute<typeof Run>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-state/api|kind=4|receiver=|name=Run", "none", false, 0, 1, "result", 0, "", "int32");
