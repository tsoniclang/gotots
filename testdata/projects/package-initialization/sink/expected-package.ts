import type { $PackageState } from "./state.js";
import { $state } from "./state.js";
import { GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function $initialize(): void {
    $state.Count = 0;
}
export { Mark, Pair, Read } from "../../../../modules/example.com/package-initialization/sink/sink.js";
export { $state };
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Count).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-initialization/sink|kind=3|receiver=|name=Count", "example.com/package-initialization/sink", "Count", "Count", "int32", true, "authored", "example.com/package-initialization/sink", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/sink/state.ts", "checked-syntax:sink.go", "23599d44db85513aa90364665cc09247974b2a6ee47bb28521bfeebfce04bfdc", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 18, 23, "go1.26", "");
attribute<typeof $initialize>().add(GoOperationFact, "gotots-go-package-initialization-fact-v2", "example.com/package-initialization/sink", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "packages/example.com/package-initialization/sink/package.ts", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 0, 1, 0, 0);
