import type { $PackageState } from "./state.js";
import { Read as Read__from_sink } from "../sink/package.js";
import { $state } from "./state.js";
import { GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function $initialize(): void {
    $state.Observed = 0;
    {
        $state.Observed = Read__from_sink();
    }
}
export { Run } from "../../../../modules/example.com/package-initialization/api/api.js";
export { $state };
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Observed).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-initialization/api|kind=3|receiver=|name=Observed", "example.com/package-initialization/api", "Observed", "Observed", "int32", true, "authored", "example.com/package-initialization/api", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/api/state.ts", "checked-syntax:api.go", "e4901dc219bf7f0ab367a170360ba53cce12443a349f7dca4ee46e110e3a90d4", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 123, 131, "go1.26", "");
attribute<typeof $initialize>().add(GoOperationFact, "gotots-go-package-initialization-fact-v2", "example.com/package-initialization/api", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/api/package.ts", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 2, 0, "example.com/package-initialization/sideeffect", 1, "example.com/package-initialization/sink", 1, 1, 0, 1, 0, "example.com/package-initialization/api|kind=3|receiver=|name=Observed", 0);
