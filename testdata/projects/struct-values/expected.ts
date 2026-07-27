import type { bool, int32 } from "../../../support/scalars.js";
export class Point {
    declare private readonly $goType: void;
    constructor(public X: int32, public Visible: bool) {
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
}
export class Box {
    declare private readonly $goType: void;
    constructor(public Point: Point, public Active: bool) {
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
}
export class Mirror {
    declare private readonly $goType: void;
    constructor(public Point: Point, public Active: bool) {
    }
}
export class Reserved {
    declare private readonly $goType: void;
    constructor(public __go_constructor: int32) {
    }
}
export class Grouped {
    declare private readonly $goType: void;
    constructor(public Left: int32, public Right: int32) {
    }
}
export class Empty {
    declare private readonly $goType: void;
    constructor() {
    }
    static $zero(): Empty {
        return new Empty();
    }
    static $equal($left: Empty, $right: Empty): bool {
        return true;
    }
}
export function NewBox(value: int32): Box {
    return new Box(new Point(value, true), value > 0);
}
export function ZeroIsFresh(): bool {
    let left = Box.$zero();
    let right = Box.$zero();
    left.Point.X = 7;
    return right.Point.X === 0;
}
export function CopyIsolated(value: Box): int32 {
    let copy = Box.$copy(value);
    copy.Point.X = copy.Point.X + 1;
    return value.Point.X * 10 + copy.Point.X;
}
export function AssignIsolated(value: Box): int32 {
    let target = Box.$zero();
    target = Box.$copy(value);
    target.Point.X = target.Point.X + 2;
    return value.Point.X * 10 + target.Point.X;
}
export function MutateParameter(value: Box): Box {
    value.Point.X = value.Point.X + 3;
    return value;
}
export function ParameterIsolated(value: Box): int32 {
    let changed = MutateParameter(Box.$copy(value));
    return value.Point.X * 10 + changed.Point.X;
}
export function Equal(left: Box, right: Box): bool {
    return Box.$equal(left, right);
}
export function Box_WithX(box: Box, value: int32): Box {
    box.Point.X = value;
    return box;
}
export function Invoke(value: Box, next: int32): Box {
    return Box_WithX(Box.$copy(value), next);
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
    return changed.Point.X * 10 + first.Point.X;
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
    const __gotots_results_0 = Duplicate(NewBox(4));
    let left = __gotots_results_0[0];
    let right = __gotots_results_0[1];
    left.Point.X = 8;
    return left.Point.X * 10 + right.Point.X;
}
export function ReadX(value: Box): int32 {
    return value.Point.X;
}
export function CompositeArgument(): int32 {
    return ReadX(new Box(new Point(6, true), true));
}
export function ReadXAfter(first: int32, value: Box): int32 {
    return first * 10 + value.Point.X;
}
export function DirectValue(): int32 {
    return 2;
}
export function CompositeSecondArgument(): int32 {
    return ReadXAfter(DirectValue(), new Box(new Point(6, true), true));
}
export function CompositeField(): int32 {
    return new Box(new Point(7, true), true).Point.X;
}
export function DirectVisible(): bool {
    return true;
}
export function DirectX(): int32 {
    return 6;
}
export function CompositeCalls(): int32 {
    return new Point(DirectX(), DirectVisible()).X;
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
    return value.Point.X * 10 + copied.Point.X;
}
export function ExplicitVarCopyResult(): int32 {
    return ExplicitVarCopy(NewBox(4));
}
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
export function GroupedResult(): int32 {
    let value = new Grouped(1, 2);
    return value.Left * 10 + value.Right;
}
export function EmptyEqual(): bool {
    let left = Empty.$zero();
    return Empty.$equal(left, new Empty());
}
