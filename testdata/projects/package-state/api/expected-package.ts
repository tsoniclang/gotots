import type { $PackageState } from "./state.js";
import { Snapshot as Snapshot__from_dep } from "../dep/package.js";
import { $state } from "./state.js";
import { GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function $initialize(): void {
    $state.Start = 0;
    {
        $state.Start = Snapshot__from_dep();
    }
}
export { Run } from "../../../../modules/example.com/package-state/api/api.js";
export { $state };
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Start).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/api|kind=3|receiver=|name=Start", "example.com/package-state/api", "Start", "Start", "int32", true, "authored", "example.com/package-state/api", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/api/state.ts", "checked-syntax:api.go", "04bc19cfcb5a31e5715d223e550c0d48217238cdd5289c10f04cddd5452b5b39", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 57, 62, "go1.26", "");
attribute<typeof $initialize>().add(GoOperationFact, "gotots-go-package-initialization-fact-v2", "example.com/package-state/api", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/api/package.ts", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 1, 0, "example.com/package-state/dep", 1, 1, 0, 1, 0, "example.com/package-state/api|kind=3|receiver=|name=Start", 0);
