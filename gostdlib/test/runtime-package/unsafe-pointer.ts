import { GoPanic } from "./panic.js";
export class GoUnsafePointer {
    declare private readonly $go$unsafePointer: void;
    private constructor() {
        GoPanic.raiseRuntime("unsafe.Pointer conversion requires an environment implementation");
    }
    static from<P>(value: P | undefined): GoUnsafePointer | undefined {
        if (value === undefined) {
            return undefined;
        }
        GoPanic.raiseRuntime("unsafe.Pointer conversion requires an environment implementation");
    }
    static to<P>(value: GoUnsafePointer | undefined): P | undefined {
        if (value === undefined) {
            return undefined;
        }
        GoPanic.raiseRuntime("unsafe.Pointer conversion requires an environment implementation");
    }
    static fromInteger<I extends number | bigint>(value: I, zero: I): GoUnsafePointer | undefined {
        if (value === zero) {
            return undefined;
        }
        GoPanic.raiseRuntime("unsafe.Pointer conversion requires an environment implementation");
    }
    static toInteger<I extends number | bigint>(value: GoUnsafePointer | undefined, zero: I): I {
        if (value === undefined) {
            return zero;
        }
        GoPanic.raiseRuntime("unsafe.Pointer conversion requires an environment implementation");
    }
}
