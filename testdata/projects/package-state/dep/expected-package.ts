import type { $PackageState } from "./state.js";
import type { int32 } from "@gotots/runtime/scalars.js";
import { Cell, mark } from "../../../../modules/example.com/package-state/dep/state.js";
import { $state } from "./state.js";
import { GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export function $initialize(): void {
    $state.A = 0;
    $state.B = 0;
    $state.Dormant = 0;
    $state.Empty = Cell.$zeroStorage();
    $state.Filled = Cell.$zeroStorage();
    $state.Trace = 0;
    $state.__go___proto__ = 0;
    $state.hidden = 0;
    {
        $state.B = mark(2);
    }
    {
        $state.A = $state.B + mark(1);
    }
    {
        $state.hidden = mark(3);
    }
    {
        $state.Filled = Cell.$storageOf(Cell.$fromStorage({
            Value: 4
        }));
    }
}
export { Cell, Cell$Storage, Snapshot } from "../../../../modules/example.com/package-state/dep/state.js";
export { $state };
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.A).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=A", "example.com/package-state/dep", "A", "A", "int32", true, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:a.go", "45ebabc98dc7aca19ab639fdb6d87f407c8879cf2f0e41340fa5674129cb3733", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 17, 18, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.B).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=B", "example.com/package-state/dep", "B", "B", "int32", true, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:b.go", "38f1f4954b7f2d12b9d9ce3e8e5d8c95113aa794aa5f666172062812bea774d6", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 17, 18, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Dormant).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=Dormant", "example.com/package-state/dep", "Dormant", "Dormant", "int32", true, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 88, 95, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Empty).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=Empty", "example.com/package-state/dep", "Empty", "Empty", "example.com/package-state/dep.Cell", true, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 106, 111, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Filled).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=Filled", "example.com/package-state/dep", "Filled", "Filled", "example.com/package-state/dep.Cell", true, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 121, 127, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.Trace).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=Trace", "example.com/package-state/dep", "Trace", "Trace", "int32", true, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 52, 57, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.__go___proto__).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=__proto__", "example.com/package-state/dep", "__proto__", "__go___proto__", "int32", false, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 68, 77, "go1.26", "");
attribute<$PackageState>().property($go$attributeTarget => $go$attributeTarget.hidden).add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "package-variable", "example.com/package-state/dep|kind=3|receiver=|name=hidden", "example.com/package-state/dep", "hidden", "hidden", "int32", false, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/state.ts", "checked-syntax:c.go", "5c589842433c4fee1804762093fbd9370586189bea5b67eed3d95e731ba56a48", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 17, 23, "go1.26", "");
attribute<typeof $initialize>().add(GoOperationFact, "gotots-go-package-initialization-fact-v2", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "packages/example.com/package-state/dep/package.ts", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 0, 8, 4, 0, 1, 0, "example.com/package-state/dep|kind=3|receiver=|name=B", 1, 1, 0, "example.com/package-state/dep|kind=3|receiver=|name=A", 2, 1, 0, "example.com/package-state/dep|kind=3|receiver=|name=hidden", 3, 1, 0, "example.com/package-state/dep|kind=3|receiver=|name=Filled", 0);
