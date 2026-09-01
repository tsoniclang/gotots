import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-initialization/sink/state.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function Mark(value: int32): int32 {
    $state.Count = $state.Count * 10 + value;
    return $state.Count;
}
attribute<typeof Mark>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-initialization/sink|kind=4|receiver=|name=Mark", "function", "example.com/package-initialization/sink", "Mark", "", "func(value int32) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/package-initialization/sink", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "modules/example.com/package-initialization/sink/sink.ts", "checked-syntax:sink.go", "23599d44db85513aa90364665cc09247974b2a6ee47bb28521bfeebfce04bfdc", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 84, 156, "go1.26", "");
attribute<typeof Mark>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-initialization/sink|kind=4|receiver=|name=Mark", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "int32");
export function Pair(): [
    int32,
    int32
] {
    return [Mark(4), Mark(5)];
}
attribute<typeof Pair>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-initialization/sink|kind=4|receiver=|name=Pair", "function", "example.com/package-initialization/sink", "Pair", "", "func() (int32, int32)|params=|results=,", "", "not-type", 0, "authored", "example.com/package-initialization/sink", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "modules/example.com/package-initialization/sink/sink.ts", "checked-syntax:sink.go", "23599d44db85513aa90364665cc09247974b2a6ee47bb28521bfeebfce04bfdc", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 158, 213, "go1.26", "");
attribute<typeof Pair>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-initialization/sink|kind=4|receiver=|name=Pair", "none", false, 0, 2, "result", 0, "", "int32", "result", 1, "", "int32");
export function Read(): int32 {
    return $state.Count;
}
attribute<typeof Read>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-initialization/sink|kind=4|receiver=|name=Read", "function", "example.com/package-initialization/sink", "Read", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/package-initialization/sink", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "modules/example.com/package-initialization/sink/sink.ts", "checked-syntax:sink.go", "23599d44db85513aa90364665cc09247974b2a6ee47bb28521bfeebfce04bfdc", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 215, 250, "go1.26", "");
attribute<typeof Read>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-initialization/sink|kind=4|receiver=|name=Read", "none", false, 0, 1, "result", 0, "", "int32");
