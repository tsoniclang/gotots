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
}
