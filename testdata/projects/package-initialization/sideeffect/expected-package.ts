import type { $PackageState } from "./state.js";
import { init } from "../../../../modules/example.com/package-initialization/sideeffect/a_init.js";
import { init__shadow_1 } from "../../../../modules/example.com/package-initialization/sideeffect/z_init.js";
import { Mark as Mark__from_sink, Pair as Pair__from_sink } from "../sink/package.js";
import { $state } from "./state.js";
import { GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function $initialize(): void {
    $state.first = 0;
    $state.second = 0;
    {
        Mark__from_sink(3);
    }
    {
        const __gotots_results_0 = Pair__from_sink();
        $state.first = __gotots_results_0[0];
        $state.second = __gotots_results_0[1];
    }
    init();
    init__shadow_1();
}
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.first).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-initialization/sideeffect|kind=3|receiver=|name=first", "example.com/package-initialization/sideeffect", "first", "first", "int32", false, "authored", "example.com/package-initialization/sideeffect", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/sideeffect/state.ts", "checked-syntax:sideeffect.go", "419f546517f5ea504afbd10dcd92484faa10316fd062708b716667b3ff0c51c8", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 95, 100, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.second).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-initialization/sideeffect|kind=3|receiver=|name=second", "example.com/package-initialization/sideeffect", "second", "second", "int32", false, "authored", "example.com/package-initialization/sideeffect", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/sideeffect/state.ts", "checked-syntax:sideeffect.go", "419f546517f5ea504afbd10dcd92484faa10316fd062708b716667b3ff0c51c8", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 102, 108, "go1.26", "");
attribute<typeof $initialize>().add(GoOperationFact, "gotots-go-package-initialization-fact-v2", "example.com/package-initialization/sideeffect", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/sideeffect/package.ts", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 1, 0, "example.com/package-initialization/sink", 2, 2, 0, 1, 0, "blank", 1, 2, 0, "example.com/package-initialization/sideeffect|kind=3|receiver=|name=first", 1, "example.com/package-initialization/sideeffect|kind=3|receiver=|name=second", 2, 0, "example.com/package-initialization/sideeffect|kind=4|receiver=|name=init", 1, "example.com/package-initialization/sideeffect|kind=4|receiver=|name=init");
