import type { int32 } from "@gotots/runtime/scalars.js";
import { Compute as Compute__from_service } from "../../../../packages/example.com/demand/service/package.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export const Compute: int32 = 5;
attribute<typeof Compute>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/demand/api|kind=1|receiver=|name=Compute", "constant", "example.com/demand/api", "Compute", "", "int32", "5", "not-type", 0, "authored", "example.com/demand/api", "example.com/demand", "", "workspace", "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b", "modules/example.com/demand/api/api.ts", "checked-syntax:api.go", "5af654ef78303af2abdb726c98846a9ea181ccb0d0b5ae28ea487dded9afc4d4", "fa2869b4a292efad0231cd02be5b7d25081cb692d8adc52e852900538c0da2be", 56, 63, "go1.26", "");
export function Run(value: int32): int32 {
    return Compute__from_service(value) + Compute;
}
attribute<typeof Run>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/demand/api|kind=4|receiver=|name=Run", "function", "example.com/demand/api", "Run", "", "func(value int32) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/demand/api", "example.com/demand", "", "workspace", "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b", "modules/example.com/demand/api/api.ts", "checked-syntax:api.go", "5af654ef78303af2abdb726c98846a9ea181ccb0d0b5ae28ea487dded9afc4d4", "fa2869b4a292efad0231cd02be5b7d25081cb692d8adc52e852900538c0da2be", 75, 147, "go1.26", "");
attribute<typeof Run>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/demand/api|kind=4|receiver=|name=Run", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "int32");
