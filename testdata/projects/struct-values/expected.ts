import type { bool, int32 } from "@gotots/runtime/scalars.js";
import type { Pointer } from "@tsonic/core/types.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { addressOf, loadPointer } from "@tsonic/core/lang.js";
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
export class Mirror {
    declare private readonly $goType: void;
    public constructor(public Point: Point, public Active: bool) {
    }
    declare private readonly then?: never;
}
export class Reserved {
    declare private readonly $goType: void;
    public constructor(public __go_constructor: int32) {
    }
    declare private readonly then?: never;
}
export class Grouped {
    declare private readonly $goType: void;
    public constructor(public Left: int32, public Right: int32) {
    }
    declare private readonly then?: never;
}
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
export function NewBox(value: int32): Box {
    const fieldValue3 = value > 0;
    const fieldValue = true;
    const fieldValue2 = value;
    const fieldValue4 = new Point(fieldValue2, fieldValue);
    return new Box(fieldValue4, fieldValue3);
}
export function Snapshot(value: Pointer<Box> | undefined): Point {
    return Point.$copy(loadPointer<Box>((value ?? GoPanic.raiseRuntime("invalid memory address or nil pointer dereference"))).Point);
}
export function ReturnSnapshotResult(): int32 {
    let value = NewBox(1);
    let snapshot = Snapshot(addressOf<Box>(value));
    value.Point.X = 2;
    return globalThis.Math.imul(snapshot.X, 10) + value.Point.X | 0;
}
export function ZeroIsFresh(): bool {
    let left = Box.$zero();
    let right = Box.$zero();
    left.Point.X = 7;
    return right.Point.X === 0;
}
export function CopyIsolated(value: Box): int32 {
    let copy = Box.$copy(value);
    copy.Point.X = copy.Point.X + 1 | 0;
    return globalThis.Math.imul(value.Point.X, 10) + copy.Point.X | 0;
}
export function AssignIsolated(value: Box): int32 {
    let target = Box.$zero();
    target = Box.$copy(value);
    target.Point.X = target.Point.X + 2 | 0;
    return globalThis.Math.imul(value.Point.X, 10) + target.Point.X | 0;
}
export function MutateParameter(value: Box): Box {
    value.Point.X = value.Point.X + 3 | 0;
    return Box.$copy(value);
}
export function ParameterIsolated(value: Box): int32 {
    let changed = MutateParameter(Box.$copy(value));
    return globalThis.Math.imul(value.Point.X, 10) + changed.Point.X | 0;
}
export function Equal(left: Box, right: Box): bool {
    return Box.$equal(left, right);
}
export function Invoke(value: Box, next: int32): Box {
    return value.WithX(next);
}
export function CopyResult(): int32 {
    return CopyIsolated(NewBox(4));
}
export function AssignResult(): int32 {
    return AssignIsolated(NewBox(4));
}
export function ParameterResult(): int32 {
    return ParameterIsolated(NewBox(4));
}
export function EqualSameResult(): bool {
    return Equal(NewBox(4), NewBox(4));
}
export function EqualDifferentResult(): bool {
    return Equal(NewBox(4), NewBox(5));
}
export function MethodResult(): int32 {
    let first = NewBox(4);
    let changed = Invoke(Box.$copy(first), 9);
    return globalThis.Math.imul(changed.Point.X, 10) + first.Point.X | 0;
}
export function ReservedValue(): int32 {
    let value = new Reserved(7);
    return value.__go_constructor;
}
export function PrimitiveZero(): bool {
    let count = 0;
    let ready = false;
    return count === 0 && ready === false;
}
export function Duplicate(value: Box): [
    Box,
    Box
] {
    return [Box.$copy(value), Box.$copy(value)];
}
export function MultipleResultIsolated(): int32 {
    const results = Duplicate(NewBox(4));
    let left = results[0];
    let right = results[1];
    left.Point.X = 8;
    return globalThis.Math.imul(left.Point.X, 10) + right.Point.X | 0;
}
export function ReadX(value: Box): int32 {
    return value.Point.X;
}
export function CompositeArgument(): int32 {
    const fieldValue7 = true;
    const fieldValue5 = true;
    const fieldValue6 = 6;
    const fieldValue8 = new Point(fieldValue6, fieldValue5);
    return ReadX(new Box(fieldValue8, fieldValue7));
}
export function ReadXAfter(first: int32, value: Box): int32 {
    return globalThis.Math.imul(first, 10) + value.Point.X | 0;
}
export function DirectValue(): int32 {
    return 2;
}
export function CompositeSecondArgument(): int32 {
    const argument = DirectValue();
    const fieldValue11 = true;
    const fieldValue9 = true;
    const fieldValue10 = 6;
    const fieldValue12 = new Point(fieldValue10, fieldValue9);
    return ReadXAfter(argument, new Box(fieldValue12, fieldValue11));
}
export function CompositeField(): int32 {
    const fieldValue15 = true;
    const fieldValue13 = true;
    const fieldValue14 = 7;
    const fieldValue16 = new Point(fieldValue14, fieldValue13);
    return new Box(fieldValue16, fieldValue15).Point.X;
}
export function DirectVisible(): bool {
    return true;
}
export function DirectX(): int32 {
    return 6;
}
export function CompositeCalls(): int32 {
    const fieldValue17 = DirectVisible();
    const fieldValue18 = DirectX();
    return new Point(fieldValue18, fieldValue17).X;
}
export function PositionalComposite(): int32 {
    let value = new Point(8, true);
    return value.X;
}
export function OmittedComposite(): bool {
    let value = new Point(5, false);
    return value.X === 5 && !value.Visible;
}
export function NotEqual(): bool {
    return !Box.$equal(NewBox(4), NewBox(5));
}
export function ExplicitVarCopy(value: Box): int32 {
    let copied = Box.$copy(value);
    copied.Point.X = 6;
    return globalThis.Math.imul(value.Point.X, 10) + copied.Point.X | 0;
}
export function ExplicitVarCopyResult(): int32 {
    return ExplicitVarCopy(NewBox(4));
}
export function ParallelAssignment(): int32 {
    let left = NewBox(4);
    let right = NewBox(9);
    const assignmentValue = Box.$copy(right);
    const assignmentValue2 = Box.$copy(left);
    left = assignmentValue;
    right = assignmentValue2;
    left.Point.X = 8;
    return globalThis.Math.imul(left.Point.X, 10) + right.Point.X | 0;
}
export function GroupedResult(): int32 {
    let value = new Grouped(1, 2);
    return globalThis.Math.imul(value.Left, 10) + value.Right | 0;
}
export function EmptyEqual(): bool {
    let left = Empty.$zero();
    return Empty.$equal(left, new Empty);
}
