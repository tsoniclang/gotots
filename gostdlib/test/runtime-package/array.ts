import type { GoStorageRegion } from "./memory-view.js";
import { goRegionRead, goRegionWrite } from "./memory-view.js";
import { GoPanic } from "./panic.js";
export class GoArray<T, N extends number | bigint> {
    private constructor(private readonly $region: GoStorageRegion<T>, public readonly length: N) {
    }
    public static zero<T, N extends number | bigint>(length: N, zero: T): GoArray<T, N> {
        const values: T[] = [];
        for (let index = 0; index < length; index++) {
            values.push(zero);
        }
        return new GoArray<T, N>({ kind: "indexed", values: values, offset: 0 }, length);
    }
    public static literal<T, N extends number | bigint>(length: N, zero: T, indexes: number[], values: T[]): GoArray<T, N> {
        if (indexes.length !== values.length) {
            GoPanic.raiseRuntime("array literal index/value length mismatch");
        }
        const result = GoArray.zero<T, N>(length, zero);
        for (let entry = 0; entry < indexes.length; entry++) {
            result.set((entry in indexes ? indexes[entry] : GoPanic.raiseRuntime("dense storage index is absent")) as number, (entry in values ? values[entry] : GoPanic.raiseRuntime("dense storage index is absent")) as T);
        }
        return result;
    }
    public copy(): GoArray<T, N> {
        if (this.$region.kind === "indexed" && (this.$region.offset === 0 && BigInt(this.$region.values.length) === BigInt(this.length))) {
            return new GoArray<T, N>({ kind: "indexed", values: Array.from(this.$region.values), offset: 0 }, this.length);
        }
        const values: T[] = [];
        for (let index = 0; index < this.length; index++) {
            values.push(this.get(index));
        }
        return new GoArray<T, N>({ kind: "indexed", values: values, offset: 0 }, this.length);
    }
    public get(index: number | bigint): T {
        return goRegionRead<T>(this.$region, this.$check(index));
    }
    public set(index: number | bigint, value: T): void {
        goRegionWrite<T>(this.$region, this.$check(index), value);
    }
    private $check(index: number | bigint): number | bigint {
        if (typeof index === "number" && !Number.isInteger(index) || (index < 0 || index >= this.length)) {
            GoPanic.raiseRuntime("array index out of bounds");
        }
        return index;
    }
    declare private readonly then?: never;
}
