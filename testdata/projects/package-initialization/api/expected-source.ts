import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-initialization/api/state.js";
import { Read as Read__from_sink } from "../../../../packages/example.com/package-initialization/sink/package.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function Run(): int32 {
    return $state.Observed + Read__from_sink();
}
attribute<typeof Run>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-initialization/api|kind=4|receiver=|name=Run", "function", "example.com/package-initialization/api", "Run", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/package-initialization/api", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "modules/example.com/package-initialization/api/api.ts", "checked-syntax:api.go", "e4901dc219bf7f0ab367a170360ba53cce12443a349f7dca4ee46e110e3a90d4", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 153, 204, "go1.26", "");
attribute<typeof Run>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-initialization/api|kind=4|receiver=|name=Run", "none", false, 0, 1, "result", 0, "", "int32");
