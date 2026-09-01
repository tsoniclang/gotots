import type { int32 } from "@gotots/runtime/scalars.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export const Offset: int32 = 2;
attribute<typeof Offset>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/demand/mathx|kind=1|receiver=|name=Offset", "constant", "example.com/demand/mathx", "Offset", "", "int32", "2", "not-type", 0, "authored", "example.com/demand/mathx", "example.com/demand", "", "workspace", "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b", "modules/example.com/demand/mathx/math.ts", "checked-syntax:math.go", "ad56d0c1fca9117570b0eb3a1d7d882da7d3bc2bd51021b3c768cbe689b32465", "fa2869b4a292efad0231cd02be5b7d25081cb692d8adc52e852900538c0da2be", 24, 30, "go1.26", "");
export function Even(value: int32): int32 {
    if (value === 0) {
        return Offset;
    }
    return Odd(value - 1);
}
attribute<typeof Even>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/demand/mathx|kind=4|receiver=|name=Even", "function", "example.com/demand/mathx", "Even", "", "func(value int32) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/demand/mathx", "example.com/demand", "", "workspace", "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b", "modules/example.com/demand/mathx/math.ts", "checked-syntax:math.go", "ad56d0c1fca9117570b0eb3a1d7d882da7d3bc2bd51021b3c768cbe689b32465", "fa2869b4a292efad0231cd02be5b7d25081cb692d8adc52e852900538c0da2be", 104, 195, "go1.26", "");
attribute<typeof Even>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/demand/mathx|kind=4|receiver=|name=Even", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "int32");
export function Odd(value: int32): int32 {
    if (value === 0) {
        return 0;
    }
    return Even(value - 1);
}
attribute<typeof Odd>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/demand/mathx|kind=4|receiver=|name=Odd", "function", "example.com/demand/mathx", "Odd", "", "func(value int32) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/demand/mathx", "example.com/demand", "", "workspace", "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b", "modules/example.com/demand/mathx/math.ts", "checked-syntax:math.go", "ad56d0c1fca9117570b0eb3a1d7d882da7d3bc2bd51021b3c768cbe689b32465", "fa2869b4a292efad0231cd02be5b7d25081cb692d8adc52e852900538c0da2be", 197, 283, "go1.26", "");
attribute<typeof Odd>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/demand/mathx|kind=4|receiver=|name=Odd", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "int32");
