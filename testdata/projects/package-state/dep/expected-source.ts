import type { int32 } from "@gotots/runtime/scalars.js";
import { $state } from "../../../../packages/example.com/package-state/dep/state.js";
import { GoAggregateFact, GoCallableFact, GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { attribute } from "@tsonic/core/lang.js";
export type Cell$Storage = {
    Value: int32;
};
export class Cell {
    declare private readonly $goType: void;
    public constructor(private readonly $storage: Cell$Storage) {
    }
    public static $storageOf($source: Cell): Cell$Storage {
        return $source.$storage;
    }
    public static $fromStorage($source: Cell$Storage): Cell {
        return new Cell($source);
    }
    public get Value(): int32 {
        return this.$storage.Value;
    }
    public set Value($value: int32) {
        this.$storage.Value = $value;
    }
    static $zeroStorage(): Cell$Storage {
        return {
            Value: 0
        };
    }
    declare private readonly then?: never;
}
attribute<Cell>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-state/dep|kind=2|receiver=|name=Cell", "type", "example.com/package-state/dep", "Cell", "", "defined=example.com/package-state/dep.Cell|underlying=struct{Value int32}", "", "defined", 0, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "modules/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 18, 46, "go1.26", "");
attribute<Cell>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/package-state/dep|kind=2|receiver=|name=Cell", "struct{Value int32}", "struct", 1);
attribute<Cell>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/package-state/dep|kind=2|receiver=|name=Cell", 0, "Value", "Value", "example.com/package-state/dep", "int32", "", false, true, false, "authored", "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "modules/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 33, 44, "go1.26", "");
attribute<typeof Cell>().method($go$attributeTarget => $go$attributeTarget.$storageOf).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/package-state/dep|kind=2|receiver=|name=Cell", "storage", "to-storage", 0);
attribute<typeof Cell>().method($go$attributeTarget => $go$attributeTarget.$fromStorage).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/package-state/dep|kind=2|receiver=|name=Cell", "storage", "from-storage", 0);
attribute<typeof Cell>().method($go$attributeTarget => $go$attributeTarget.$zeroStorage).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/package-state/dep|kind=2|receiver=|name=Cell", "storage-zero", "storage-zero", 0);
export function mark(value: int32): int32 {
    $state.Trace = $state.Trace * 10 + value;
    $state.__go___proto__++;
    return value;
}
attribute<typeof mark>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-state/dep|kind=4|receiver=|name=mark", "function", "example.com/package-state/dep", "mark", "", "func(value int32) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "modules/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 151, 236, "go1.26", "");
attribute<typeof mark>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-state/dep|kind=4|receiver=|name=mark", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "int32");
export function Snapshot(): int32 {
    return $state.A * 10000 + $state.B * 1000 + $state.Trace * 10 + $state.hidden + $state.__go___proto__ + Cell.$storageOf(Cell.$fromStorage($state.Empty)).Value + Cell.$storageOf(Cell.$fromStorage($state.Filled)).Value;
}
attribute<typeof Snapshot>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/package-state/dep|kind=4|receiver=|name=Snapshot", "function", "example.com/package-state/dep", "Snapshot", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/package-state/dep", "example.com/package-state", "", "workspace", "30b189d6a1f032298fdb86d2246d93c1095afdffc5c48f3a05e48e2d7fd2d1f6", "modules/example.com/package-state/dep/state.ts", "checked-syntax:state.go", "6cbe80466c14135c0b39033bcd0bcef8dfe9876a189c5ab5d90d7b889990063e", "1bfd6e908f05dc75350012d93b8d28f82d6ee0d0a497d2ed55f07f80c921addc", 238, 361, "go1.26", "");
attribute<typeof Snapshot>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/package-state/dep|kind=4|receiver=|name=Snapshot", "none", false, 0, 1, "result", 0, "", "int32");
