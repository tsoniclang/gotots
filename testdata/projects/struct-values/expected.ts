import type { bool, int32 } from "@gotots/runtime/scalars.js";
import type { Pointer } from "@tsonic/core/types.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { GoAggregateFact, GoCallableFact, GoDeclarationFact, GoOperationFact } from "@gotots/runtime/source-fact.js";
import { addressOf, attribute, loadPointer } from "@tsonic/core/lang.js";
export class Point {
    declare private readonly $goType: void;
    public constructor(public X: int32, public Visible: bool) {
    }
    static $zero(): Point {
        return new Point(0, false);
    }
    static $copy($source: Point): Point {
        return new Point($source.X, $source.Visible);
    }
    static $equal($left: Point, $right: Point): bool {
        return $left.X === $right.X && $left.Visible === $right.Visible;
    }
    declare private readonly then?: never;
}
attribute<Point>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Point", "type", "example.com/structvalues", "Point", "", "defined=example.com/structvalues.Point|underlying=struct{X int32; Visible bool}", "", "defined", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 27, 146, "go1.26", "");
attribute<Point>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Point", "struct{X int32; Visible bool}", "struct", 2);
attribute<Point>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Point", 0, "X", "X", "example.com/structvalues", "int32", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 117, 130, "go1.26", "");
attribute<Point>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Point", 1, "Visible", "Visible", "example.com/structvalues", "bool", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 132, 144, "go1.26", "");
attribute<typeof Point>().method($go$attributeTarget => $go$attributeTarget.$zero).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Point", "zero", "zero", 0);
attribute<typeof Point>().method($go$attributeTarget => $go$attributeTarget.$copy).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Point", "copy", "copy", 0);
attribute<typeof Point>().method($go$attributeTarget => $go$attributeTarget.$equal).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Point", "equal", "equal", 0);
export class Box {
    declare private readonly $goType: void;
    public constructor(public Point: Point, public Active: bool) {
    }
    static $zero(): Box {
        return new Box(Point.$zero(), false);
    }
    static $copy($source: Box): Box {
        return new Box(Point.$copy($source.Point), $source.Active);
    }
    static $equal($left: Box, $right: Box): bool {
        return Point.$equal($left.Point, $right.Point) && $left.Active === $right.Active;
    }
    declare private readonly then?: never;
    WithX(value: int32): Box {
        let box: Box = Box.$copy(this);
        box.Point.X = value;
        return Box.$copy(box);
    }
}
attribute<Box>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Box", "type", "example.com/structvalues", "Box", "", "defined=example.com/structvalues.Box|underlying=struct{Point example.com/structvalues.Point; Active bool}", "", "defined", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 153, 194, "go1.26", "");
attribute<Box>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Box", "struct{Point example.com/structvalues.Point; Active bool}", "struct", 2);
attribute<Box>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Box", 0, "Point", "Point", "example.com/structvalues", "example.com/structvalues.Point", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 167, 179, "go1.26", "");
attribute<Box>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Box", 1, "Active", "Active", "example.com/structvalues", "bool", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 181, 192, "go1.26", "");
attribute<typeof Box>().method($go$attributeTarget => $go$attributeTarget.$zero).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Box", "zero", "zero", 0);
attribute<typeof Box>().method($go$attributeTarget => $go$attributeTarget.$copy).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Box", "copy", "copy", 0);
attribute<typeof Box>().method($go$attributeTarget => $go$attributeTarget.$equal).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Box", "equal", "equal", 0);
attribute<Box>().method($go$attributeTarget => $go$attributeTarget.WithX).add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=example.com/structvalues.Box|name=WithX", "function", "example.com/structvalues", "WithX", "example.com/structvalues.Box", "func(value int32) example.com/structvalues.Box|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1346, 1420, "go1.26", "");
attribute<Box>().method($go$attributeTarget => $go$attributeTarget.WithX).add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=example.com/structvalues.Box|name=WithX", "value", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "example.com/structvalues.Box");
export class Mirror {
    declare private readonly $goType: void;
    public constructor(public Point: Point, public Active: bool) {
    }
    declare private readonly then?: never;
}
attribute<Mirror>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Mirror", "type", "example.com/structvalues", "Mirror", "", "defined=example.com/structvalues.Mirror|underlying=struct{Point example.com/structvalues.Point; Active bool}", "", "defined", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 201, 245, "go1.26", "");
attribute<Mirror>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Mirror", "struct{Point example.com/structvalues.Point; Active bool}", "struct", 2);
attribute<Mirror>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Mirror", 0, "Point", "Point", "example.com/structvalues", "example.com/structvalues.Point", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 218, 230, "go1.26", "");
attribute<Mirror>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Mirror", 1, "Active", "Active", "example.com/structvalues", "bool", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 232, 243, "go1.26", "");
export class Reserved {
    declare private readonly $goType: void;
    public constructor(public __go_constructor: int32) {
    }
    declare private readonly then?: never;
}
attribute<Reserved>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Reserved", "type", "example.com/structvalues", "Reserved", "", "defined=example.com/structvalues.Reserved|underlying=struct{constructor int32}", "", "defined", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 252, 290, "go1.26", "");
attribute<Reserved>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Reserved", "struct{constructor int32}", "struct", 1);
attribute<Reserved>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Reserved", 0, "constructor", "__go_constructor", "example.com/structvalues", "int32", "", false, false, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 271, 288, "go1.26", "");
export class Grouped {
    declare private readonly $goType: void;
    public constructor(public Left: int32, public Right: int32) {
    }
    declare private readonly then?: never;
}
attribute<Grouped>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Grouped", "type", "example.com/structvalues", "Grouped", "", "defined=example.com/structvalues.Grouped|underlying=struct{Left int32; Right int32}", "", "defined", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 297, 334, "go1.26", "");
attribute<Grouped>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Grouped", "struct{Left int32; Right int32}", "struct", 2);
attribute<Grouped>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Grouped", 0, "Left", "Left", "example.com/structvalues", "int32", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 315, 332, "go1.26", "");
attribute<Grouped>().add(GoDeclarationFact, "gotots-go-source-member-fact-v3", "field", "example.com/structvalues|kind=2|receiver=|name=Grouped", 1, "Right", "Right", "example.com/structvalues", "int32", "", false, true, false, "authored", "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 315, 332, "go1.26", "");
export class Empty {
    declare private readonly $goType: void;
    public constructor() {
    }
    static $zero(): Empty {
        return new Empty();
    }
    static $equal($left: Empty, $right: Empty): bool {
        return true;
    }
    declare private readonly then?: never;
}
attribute<Empty>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Empty", "type", "example.com/structvalues", "Empty", "", "defined=example.com/structvalues.Empty|underlying=struct{}", "", "defined", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 341, 355, "go1.26", "");
attribute<Empty>().add(GoAggregateFact, "gotots-go-source-aggregate-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Empty", "struct{}", "struct", 0);
attribute<typeof Empty>().method($go$attributeTarget => $go$attributeTarget.$zero).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Empty", "zero", "zero", 0);
attribute<typeof Empty>().method($go$attributeTarget => $go$attributeTarget.$equal).add(GoOperationFact, "gotots-go-struct-operation-fact-v1", "example.com/structvalues|kind=2|receiver=|name=Empty", "equal", "equal", 0);
export function NewBox(value: int32): Box {
    const __gotots_field_2 = value > 0;
    const __gotots_field_0 = true;
    const __gotots_field_1 = value;
    const __gotots_field_3 = new Point(__gotots_field_1, __gotots_field_0);
    return new Box(__gotots_field_3, __gotots_field_2);
}
attribute<typeof NewBox>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=NewBox", "function", "example.com/structvalues", "NewBox", "", "func(value int32) example.com/structvalues.Box|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 357, 484, "go1.26", "");
attribute<typeof NewBox>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=NewBox", "none", false, 1, 1, "parameter", 0, "value", "int32", "result", 0, "", "example.com/structvalues.Box");
export function Snapshot(value: Pointer<Box> | undefined): Point {
    return Point.$copy(loadPointer<Box>((value ?? GoPanic.raiseRuntime("invalid memory address or nil pointer dereference"))).Point);
}
attribute<typeof Snapshot>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Snapshot", "function", "example.com/structvalues", "Snapshot", "", "func(value *example.com/structvalues.Box) example.com/structvalues.Point|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 486, 541, "go1.26", "");
attribute<typeof Snapshot>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Snapshot", "none", false, 1, 1, "parameter", 0, "value", "*example.com/structvalues.Box", "result", 0, "", "example.com/structvalues.Point");
export function ReturnSnapshotResult(): int32 {
    let value = NewBox(1);
    let snapshot = Snapshot(addressOf<Box>(value));
    value.Point.X = 2;
    return snapshot.X * 10 + value.Point.X;
}
attribute<typeof ReturnSnapshotResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReturnSnapshotResult", "function", "example.com/structvalues", "ReturnSnapshotResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 543, 687, "go1.26", "");
attribute<typeof ReturnSnapshotResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReturnSnapshotResult", "none", false, 0, 1, "result", 0, "", "int32");
export function ZeroIsFresh(): bool {
    let left = Box.$zero();
    let right = Box.$zero();
    left.Point.X = 7;
    return right.Point.X === 0;
}
attribute<typeof ZeroIsFresh>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ZeroIsFresh", "function", "example.com/structvalues", "ZeroIsFresh", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 689, 790, "go1.26", "");
attribute<typeof ZeroIsFresh>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ZeroIsFresh", "none", false, 0, 1, "result", 0, "", "bool");
export function CopyIsolated(value: Box): int32 {
    let copy = Box.$copy(value);
    copy.Point.X = copy.Point.X + 1;
    return value.Point.X * 10 + copy.Point.X;
}
attribute<typeof CopyIsolated>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CopyIsolated", "function", "example.com/structvalues", "CopyIsolated", "", "func(value example.com/structvalues.Box) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 792, 918, "go1.26", "");
attribute<typeof CopyIsolated>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CopyIsolated", "none", false, 1, 1, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "int32");
export function AssignIsolated(value: Box): int32 {
    let target = Box.$zero();
    target = Box.$copy(value);
    target.Point.X = target.Point.X + 2;
    return value.Point.X * 10 + target.Point.X;
}
attribute<typeof AssignIsolated>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=AssignIsolated", "function", "example.com/structvalues", "AssignIsolated", "", "func(value example.com/structvalues.Box) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 920, 1071, "go1.26", "");
attribute<typeof AssignIsolated>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=AssignIsolated", "none", false, 1, 1, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "int32");
export function MutateParameter(value: Box): Box {
    value.Point.X = value.Point.X + 3;
    return Box.$copy(value);
}
attribute<typeof MutateParameter>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=MutateParameter", "function", "example.com/structvalues", "MutateParameter", "", "func(value example.com/structvalues.Box) example.com/structvalues.Box|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1073, 1161, "go1.26", "");
attribute<typeof MutateParameter>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=MutateParameter", "none", false, 1, 1, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "example.com/structvalues.Box");
export function ParameterIsolated(value: Box): int32 {
    let changed = MutateParameter(Box.$copy(value));
    return value.Point.X * 10 + changed.Point.X;
}
attribute<typeof ParameterIsolated>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ParameterIsolated", "function", "example.com/structvalues", "ParameterIsolated", "", "func(value example.com/structvalues.Box) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1163, 1284, "go1.26", "");
attribute<typeof ParameterIsolated>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ParameterIsolated", "none", false, 1, 1, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "int32");
export function Equal(left: Box, right: Box): bool {
    return Box.$equal(left, right);
}
attribute<typeof Equal>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Equal", "function", "example.com/structvalues", "Equal", "", "func(left example.com/structvalues.Box, right example.com/structvalues.Box) bool|params=left,right|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1286, 1344, "go1.26", "");
attribute<typeof Equal>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Equal", "none", false, 2, 1, "parameter", 0, "left", "example.com/structvalues.Box", "parameter", 1, "right", "example.com/structvalues.Box", "result", 0, "", "bool");
export function Invoke(value: Box, next: int32): Box {
    return value.WithX(next);
}
attribute<typeof Invoke>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Invoke", "function", "example.com/structvalues", "Invoke", "", "func(value example.com/structvalues.Box, next int32) example.com/structvalues.Box|params=value,next|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1422, 1490, "go1.26", "");
attribute<typeof Invoke>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Invoke", "none", false, 2, 1, "parameter", 0, "value", "example.com/structvalues.Box", "parameter", 1, "next", "int32", "result", 0, "", "example.com/structvalues.Box");
export function CopyResult(): int32 {
    return CopyIsolated(NewBox(4));
}
attribute<typeof CopyResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CopyResult", "function", "example.com/structvalues", "CopyResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1492, 1551, "go1.26", "");
attribute<typeof CopyResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CopyResult", "none", false, 0, 1, "result", 0, "", "int32");
export function AssignResult(): int32 {
    return AssignIsolated(NewBox(4));
}
attribute<typeof AssignResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=AssignResult", "function", "example.com/structvalues", "AssignResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1553, 1616, "go1.26", "");
attribute<typeof AssignResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=AssignResult", "none", false, 0, 1, "result", 0, "", "int32");
export function ParameterResult(): int32 {
    return ParameterIsolated(NewBox(4));
}
attribute<typeof ParameterResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ParameterResult", "function", "example.com/structvalues", "ParameterResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1618, 1687, "go1.26", "");
attribute<typeof ParameterResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ParameterResult", "none", false, 0, 1, "result", 0, "", "int32");
export function EqualSameResult(): bool {
    return Equal(NewBox(4), NewBox(4));
}
attribute<typeof EqualSameResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=EqualSameResult", "function", "example.com/structvalues", "EqualSameResult", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1689, 1756, "go1.26", "");
attribute<typeof EqualSameResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=EqualSameResult", "none", false, 0, 1, "result", 0, "", "bool");
export function EqualDifferentResult(): bool {
    return Equal(NewBox(4), NewBox(5));
}
attribute<typeof EqualDifferentResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=EqualDifferentResult", "function", "example.com/structvalues", "EqualDifferentResult", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1758, 1830, "go1.26", "");
attribute<typeof EqualDifferentResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=EqualDifferentResult", "none", false, 0, 1, "result", 0, "", "bool");
export function MethodResult(): int32 {
    let first = NewBox(4);
    let changed = Invoke(Box.$copy(first), 9);
    return changed.Point.X * 10 + first.Point.X;
}
attribute<typeof MethodResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=MethodResult", "function", "example.com/structvalues", "MethodResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1832, 1953, "go1.26", "");
attribute<typeof MethodResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=MethodResult", "none", false, 0, 1, "result", 0, "", "int32");
export function ReservedValue(): int32 {
    let value = new Reserved(7);
    return value.__go_constructor;
}
attribute<typeof ReservedValue>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReservedValue", "function", "example.com/structvalues", "ReservedValue", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 1955, 2046, "go1.26", "");
attribute<typeof ReservedValue>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReservedValue", "none", false, 0, 1, "result", 0, "", "int32");
export function PrimitiveZero(): bool {
    let count = 0;
    let ready = false;
    return count === 0 && ready === false;
}
attribute<typeof PrimitiveZero>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=PrimitiveZero", "function", "example.com/structvalues", "PrimitiveZero", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2048, 2147, "go1.26", "");
attribute<typeof PrimitiveZero>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=PrimitiveZero", "none", false, 0, 1, "result", 0, "", "bool");
export function Duplicate(value: Box): [
    Box,
    Box
] {
    return [Box.$copy(value), Box.$copy(value)];
}
attribute<typeof Duplicate>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Duplicate", "function", "example.com/structvalues", "Duplicate", "", "func(value example.com/structvalues.Box) (example.com/structvalues.Box, example.com/structvalues.Box)|params=value|results=,", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2149, 2210, "go1.26", "");
attribute<typeof Duplicate>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=Duplicate", "none", false, 1, 2, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "example.com/structvalues.Box", "result", 1, "", "example.com/structvalues.Box");
export function MultipleResultIsolated(): int32 {
    const __gotots_results_0 = Duplicate(NewBox(4));
    let left = __gotots_results_0[0];
    let right = __gotots_results_0[1];
    left.Point.X = 8;
    return left.Point.X * 10 + right.Point.X;
}
attribute<typeof MultipleResultIsolated>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=MultipleResultIsolated", "function", "example.com/structvalues", "MultipleResultIsolated", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2212, 2346, "go1.26", "");
attribute<typeof MultipleResultIsolated>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=MultipleResultIsolated", "none", false, 0, 1, "result", 0, "", "int32");
export function ReadX(value: Box): int32 {
    return value.Point.X;
}
attribute<typeof ReadX>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReadX", "function", "example.com/structvalues", "ReadX", "", "func(value example.com/structvalues.Box) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2348, 2401, "go1.26", "");
attribute<typeof ReadX>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReadX", "none", false, 1, 1, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "int32");
export function CompositeArgument(): int32 {
    const __gotots_field_6 = true;
    const __gotots_field_4 = true;
    const __gotots_field_5 = 6;
    const __gotots_field_7 = new Point(__gotots_field_5, __gotots_field_4);
    const __gotots_argument_0 = new Box(__gotots_field_7, __gotots_field_6);
    return ReadX(__gotots_argument_0);
}
attribute<typeof CompositeArgument>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeArgument", "function", "example.com/structvalues", "CompositeArgument", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2403, 2530, "go1.26", "");
attribute<typeof CompositeArgument>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeArgument", "none", false, 0, 1, "result", 0, "", "int32");
export function ReadXAfter(first: int32, value: Box): int32 {
    return first * 10 + value.Point.X;
}
attribute<typeof ReadXAfter>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReadXAfter", "function", "example.com/structvalues", "ReadXAfter", "", "func(first int32, value example.com/structvalues.Box) int32|params=first,value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2532, 2614, "go1.26", "");
attribute<typeof ReadXAfter>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ReadXAfter", "none", false, 2, 1, "parameter", 0, "first", "int32", "parameter", 1, "value", "example.com/structvalues.Box", "result", 0, "", "int32");
export function DirectValue(): int32 {
    return 2;
}
attribute<typeof DirectValue>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=DirectValue", "function", "example.com/structvalues", "DirectValue", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2616, 2654, "go1.26", "");
attribute<typeof DirectValue>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=DirectValue", "none", false, 0, 1, "result", 0, "", "int32");
export function CompositeSecondArgument(): int32 {
    const __gotots_argument_1 = DirectValue();
    const __gotots_field_10 = true;
    const __gotots_field_8 = true;
    const __gotots_field_9 = 6;
    const __gotots_field_11 = new Point(__gotots_field_9, __gotots_field_8);
    const __gotots_argument_2 = new Box(__gotots_field_11, __gotots_field_10);
    return ReadXAfter(__gotots_argument_1, __gotots_argument_2);
}
attribute<typeof CompositeSecondArgument>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeSecondArgument", "function", "example.com/structvalues", "CompositeSecondArgument", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2656, 2809, "go1.26", "");
attribute<typeof CompositeSecondArgument>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeSecondArgument", "none", false, 0, 1, "result", 0, "", "int32");
export function CompositeField(): int32 {
    const __gotots_field_14 = true;
    const __gotots_field_12 = true;
    const __gotots_field_13 = 7;
    const __gotots_field_15 = new Point(__gotots_field_13, __gotots_field_12);
    return new Box(__gotots_field_15, __gotots_field_14).Point.X;
}
attribute<typeof CompositeField>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeField", "function", "example.com/structvalues", "CompositeField", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2811, 2936, "go1.26", "");
attribute<typeof CompositeField>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeField", "none", false, 0, 1, "result", 0, "", "int32");
export function DirectVisible(): bool {
    return true;
}
attribute<typeof DirectVisible>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=DirectVisible", "function", "example.com/structvalues", "DirectVisible", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2938, 2980, "go1.26", "");
attribute<typeof DirectVisible>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=DirectVisible", "none", false, 0, 1, "result", 0, "", "bool");
export function DirectX(): int32 {
    return 6;
}
attribute<typeof DirectX>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=DirectX", "function", "example.com/structvalues", "DirectX", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 2982, 3016, "go1.26", "");
attribute<typeof DirectX>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=DirectX", "none", false, 0, 1, "result", 0, "", "int32");
export function CompositeCalls(): int32 {
    const __gotots_field_16 = DirectVisible();
    const __gotots_field_17 = DirectX();
    return new Point(__gotots_field_17, __gotots_field_16).X;
}
attribute<typeof CompositeCalls>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeCalls", "function", "example.com/structvalues", "CompositeCalls", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3018, 3119, "go1.26", "");
attribute<typeof CompositeCalls>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=CompositeCalls", "none", false, 0, 1, "result", 0, "", "int32");
export function PositionalComposite(): int32 {
    let value = new Point(8, true);
    return value.X;
}
attribute<typeof PositionalComposite>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=PositionalComposite", "function", "example.com/structvalues", "PositionalComposite", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3121, 3198, "go1.26", "");
attribute<typeof PositionalComposite>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=PositionalComposite", "none", false, 0, 1, "result", 0, "", "int32");
export function OmittedComposite(): bool {
    let value = new Point(5, false);
    return value.X === 5 && !value.Visible;
}
attribute<typeof OmittedComposite>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=OmittedComposite", "function", "example.com/structvalues", "OmittedComposite", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3200, 3293, "go1.26", "");
attribute<typeof OmittedComposite>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=OmittedComposite", "none", false, 0, 1, "result", 0, "", "bool");
export function NotEqual(): bool {
    return !Box.$equal(NewBox(4), NewBox(5));
}
attribute<typeof NotEqual>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=NotEqual", "function", "example.com/structvalues", "NotEqual", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3295, 3350, "go1.26", "");
attribute<typeof NotEqual>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=NotEqual", "none", false, 0, 1, "result", 0, "", "bool");
export function ExplicitVarCopy(value: Box): int32 {
    let copied = Box.$copy(value);
    copied.Point.X = 6;
    return value.Point.X * 10 + copied.Point.X;
}
attribute<typeof ExplicitVarCopy>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ExplicitVarCopy", "function", "example.com/structvalues", "ExplicitVarCopy", "", "func(value example.com/structvalues.Box) int32|params=value|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3352, 3479, "go1.26", "");
attribute<typeof ExplicitVarCopy>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ExplicitVarCopy", "none", false, 1, 1, "parameter", 0, "value", "example.com/structvalues.Box", "result", 0, "", "int32");
export function ExplicitVarCopyResult(): int32 {
    return ExplicitVarCopy(NewBox(4));
}
attribute<typeof ExplicitVarCopyResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ExplicitVarCopyResult", "function", "example.com/structvalues", "ExplicitVarCopyResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3481, 3554, "go1.26", "");
attribute<typeof ExplicitVarCopyResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ExplicitVarCopyResult", "none", false, 0, 1, "result", 0, "", "int32");
export function ParallelAssignment(): int32 {
    let left = NewBox(4);
    let right = NewBox(9);
    const __gotots_assign_0 = Box.$copy(right);
    const __gotots_assign_1 = Box.$copy(left);
    left = __gotots_assign_0;
    right = __gotots_assign_1;
    left.Point.X = 8;
    return left.Point.X * 10 + right.Point.X;
}
attribute<typeof ParallelAssignment>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ParallelAssignment", "function", "example.com/structvalues", "ParallelAssignment", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3556, 3715, "go1.26", "");
attribute<typeof ParallelAssignment>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=ParallelAssignment", "none", false, 0, 1, "result", 0, "", "int32");
export function GroupedResult(): int32 {
    let value = new Grouped(1, 2);
    return value.Left * 10 + value.Right;
}
attribute<typeof GroupedResult>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=GroupedResult", "function", "example.com/structvalues", "GroupedResult", "", "func() int32|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3717, 3807, "go1.26", "");
attribute<typeof GroupedResult>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=GroupedResult", "none", false, 0, 1, "result", 0, "", "int32");
export function EmptyEqual(): bool {
    let left = Empty.$zero();
    return Empty.$equal(left, new Empty);
}
attribute<typeof EmptyEqual>().add(GoDeclarationFact, "gotots-go-source-declaration-fact-v1", "example.com/structvalues|kind=4|receiver=|name=EmptyEqual", "function", "example.com/structvalues", "EmptyEqual", "", "func() bool|params=|results=", "", "not-type", 0, "authored", "example.com/structvalues", "example.com/structvalues", "", "workspace", "4080345a94e49de99c2c3348898be19977546087c0a2f2a14d145f8705dfa38a", "modules/example.com/structvalues/_root/source.ts", "checked-syntax:source.go", "bfcb0e39086f4b5148523afcf055ab7b951840ad297ae782c1f44169282a110b", "89b2b72bcc6a6f832fd10d023f26fc28b9515a16518c23aa23180684bd728845", 3809, 3875, "go1.26", "");
attribute<typeof EmptyEqual>().add(GoCallableFact, "gotots-go-source-callable-fact-v1", "example.com/structvalues|kind=4|receiver=|name=EmptyEqual", "none", false, 0, 1, "result", 0, "", "bool");
