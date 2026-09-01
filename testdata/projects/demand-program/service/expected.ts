import type { int32 } from "@gotots/runtime/scalars.js";
import { Even as Even__from_mathx } from "../../../../packages/example.com/demand/mathx/package.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function Compute(value: int32): int32 {
    let Even = value;
    Even += Even__from_mathx(value);
    return Even;
}
attribute<typeof Compute>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/demand/service|kind=4|receiver=|name=Compute", "function", "example.com/demand/service", "Compute", "", "func(value int32) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/demand/service", "example.com/demand", "", "workspace", "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b", "modules/example.com/demand/service/service.ts", "checked-syntax:service.go", "2c6a6fc3ed92fa0b44b3442200396dc811396863e0ccf06348eb62e182f0c49e", "fa2869b4a292efad0231cd02be5b7d25081cb692d8adc52e852900538c0da2be", 52, 142, "go1.26", "");
attribute<typeof Compute>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/demand/service|kind=4|receiver=|name=Compute", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "int32");
