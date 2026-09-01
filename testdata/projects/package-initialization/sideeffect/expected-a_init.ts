import { Mark as Mark__from_sink } from "../../../../packages/example.com/package-initialization/sink/package.js";
import { GoCallableFact, GoDeclarationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function init(): void {
    Mark__from_sink(6);
}
attribute<typeof init>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-initialization/sideeffect|kind=4|receiver=|name=init", "function", "example.com/package-initialization/sideeffect", "init", "", "func()|params=|results=", "", "not-type", 0, "authored", "example.com/package-initialization/sideeffect", "example.com/package-initialization", "", "workspace", "f6d94e9e1887a5f5bbdf372dbf368ae6a7af306c630de375a487f84948e943af", "modules/example.com/package-initialization/sideeffect/a_init.ts", "checked-syntax:a_init.go", "1613c917acfcc76d6eba6ab4357b837e04e21cb9abd73cb630236ceca4d75aa2", "90e111ba62574d08dc1c9677dea7c643dfd558d05351bc197f0b42f7eba324b0", 70, 99, "go1.26", "");
attribute<typeof init>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-initialization/sideeffect|kind=4|receiver=|name=init", "none", false, 0, 0);
