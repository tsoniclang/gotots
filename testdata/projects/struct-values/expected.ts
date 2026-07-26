import type { bool, int32 } from "../../../support/scalars.js";
export class Point {
    declare private readonly $goType: void;
    constructor(public X: int32, public Visible: bool) {
    }
    static $zero(): Point {
        return new Point(0 as int32, false as bool);
    }
    static $copy($source: Point): Point {
        return new Point($source.X, $source.Visible);
    }
    static $assign($target: Point, $source: Point): void {
        $target.X = $source.X;
        $target.Visible = $source.Visible;
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
        return new Box(Point.$zero(), false as bool);
    }
    static $copy($source: Box): Box {
        return new Box(Point.$copy($source.Point), $source.Active);
    }
    static $assign($target: Box, $source: Box): void {
        Point.$assign($target.Point, $source.Point);
        $target.Active = $source.Active;
    }
    static $equal($left: Box, $right: Box): bool {
        return Point.$equal($left.Point, $right.Point) && $left.Active === $right.Active;
    }
}
export class Mirror {
    declare private readonly $goType: void;
    constructor(public Point: Point, public Active: bool) {
    }
    static $zero(): Mirror {
        return new Mirror(Point.$zero(), false as bool);
    }
    static $copy($source: Mirror): Mirror {
        return new Mirror(Point.$copy($source.Point), $source.Active);
    }
    static $assign($target: Mirror, $source: Mirror): void {
        Point.$assign($target.Point, $source.Point);
        $target.Active = $source.Active;
    }
    static $equal($left: Mirror, $right: Mirror): bool {
        return Point.$equal($left.Point, $right.Point) && $left.Active === $right.Active;
    }
}
export class Reserved {
    declare private readonly $goType: void;
    constructor(public __go_constructor: int32) {
    }
    static $zero(): Reserved {
        return new Reserved(0 as int32);
    }
    static $copy($source: Reserved): Reserved {
        return new Reserved($source.__go_constructor);
    }
    static $assign($target: Reserved, $source: Reserved): void {
        $target.__go_constructor = $source.__go_constructor;
    }
    static $equal($left: Reserved, $right: Reserved): bool {
        return $left.__go_constructor === $right.__go_constructor;
    }
}
export function NewBox(value: int32): Box {
    const __gotots_field_2: bool = value > (0 as int32);
    const __gotots_field_0: bool = true as bool;
    const __gotots_field_1: int32 = value;
    const __gotots_field_3: Point = new Point(__gotots_field_1, __gotots_field_0);
    return new Box(__gotots_field_3, __gotots_field_2);
}
export function ZeroIsFresh(): bool {
    let left: Box = Box.$zero();
    let right: Box = Box.$zero();
    left.Point.X = 7 as int32;
    return right.Point.X === 0 as int32;
}
export function CopyIsolated(value: Box): int32 {
    let copy: Box = Box.$copy(value);
    copy.Point.X = (copy.Point.X + (1 as int32)) | 0;
    return (Math.imul(value.Point.X, 10 as int32) + copy.Point.X) | 0;
}
export function AssignIsolated(value: Box): int32 {
    let target: Box = Box.$zero();
    Box.$assign(target, value);
    target.Point.X = (target.Point.X + (2 as int32)) | 0;
    return (Math.imul(value.Point.X, 10 as int32) + target.Point.X) | 0;
}
export function MutateParameter(value: Box): Box {
    value.Point.X = (value.Point.X + (3 as int32)) | 0;
    return value;
}
export function ParameterIsolated(value: Box): int32 {
    let changed: Box = MutateParameter(Box.$copy(value));
    return (Math.imul(value.Point.X, 10 as int32) + changed.Point.X) | 0;
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
    return CopyIsolated(NewBox(4 as int32));
}
export function AssignResult(): int32 {
    return AssignIsolated(NewBox(4 as int32));
}
export function ParameterResult(): int32 {
    return ParameterIsolated(NewBox(4 as int32));
}
export function EqualSameResult(): bool {
    return Equal(NewBox(4 as int32), NewBox(4 as int32));
}
export function EqualDifferentResult(): bool {
    return Equal(NewBox(4 as int32), NewBox(5 as int32));
}
export function MethodResult(): int32 {
    let first: Box = NewBox(4 as int32);
    let changed: Box = Invoke(Box.$copy(first), 9 as int32);
    return (Math.imul(changed.Point.X, 10 as int32) + first.Point.X) | 0;
}
export function ReservedValue(): int32 {
    let value: Reserved = new Reserved(7 as int32);
    return value.__go_constructor;
}
export function PrimitiveZero(): bool {
    let count: int32 = 0 as int32;
    let ready: bool = false as bool;
    return count === 0 as int32 && ready === false as bool;
}
export function Duplicate(value: Box): [
    Box,
    Box
] {
    return [Box.$copy(value), Box.$copy(value)];
}
export function MultipleResultIsolated(): int32 {
    const __gotots_results_0: [
        Box,
        Box
    ] = Duplicate(NewBox(4 as int32));
    let left: Box = __gotots_results_0[0];
    let right: Box = __gotots_results_0[1];
    left.Point.X = 8 as int32;
    return (Math.imul(left.Point.X, 10 as int32) + right.Point.X) | 0;
}
export function ReadX(value: Box): int32 {
    return value.Point.X;
}
export function CompositeArgument(): int32 {
    const __gotots_field_6: bool = true as bool;
    const __gotots_field_4: bool = true as bool;
    const __gotots_field_5: int32 = 6 as int32;
    const __gotots_field_7: Point = new Point(__gotots_field_5, __gotots_field_4);
    const __gotots_argument_0: Box = new Box(__gotots_field_7, __gotots_field_6);
    return ReadX(__gotots_argument_0);
}
export function ReadXAfter(first: int32, value: Box): int32 {
    return (Math.imul(first, 10 as int32) + value.Point.X) | 0;
}
export function DirectValue(): int32 {
    return 2 as int32;
}
export function CompositeSecondArgument(): int32 {
    const __gotots_argument_1: int32 = DirectValue();
    const __gotots_field_10: bool = true as bool;
    const __gotots_field_8: bool = true as bool;
    const __gotots_field_9: int32 = 6 as int32;
    const __gotots_field_11: Point = new Point(__gotots_field_9, __gotots_field_8);
    const __gotots_argument_2: Box = new Box(__gotots_field_11, __gotots_field_10);
    return ReadXAfter(__gotots_argument_1, __gotots_argument_2);
}
export function CompositeField(): int32 {
    const __gotots_field_14: bool = true as bool;
    const __gotots_field_12: bool = true as bool;
    const __gotots_field_13: int32 = 7 as int32;
    const __gotots_field_15: Point = new Point(__gotots_field_13, __gotots_field_12);
    return new Box(__gotots_field_15, __gotots_field_14).Point.X;
}
