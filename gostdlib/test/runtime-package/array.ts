import { GoDenseIndex } from "./dense-index.js";
import { GoPanic } from "./panic.js";
export class GoArray<T, N extends number> {
    private constructor(private readonly $values: T[], private readonly $offset: number, public readonly length: N) {
    }
    public static zero<T, N extends number>(length: N, zero: T): GoArray<T, N> {
        const values: T[] = [];
        for (let index = 0; index < length; index++) {
            values.push(zero);
        }
        return new GoArray<T, N>(values, 0, length);
    }
    public static literal<T, N extends number>(length: N, zero: T, indexes: number[], values: T[]): GoArray<T, N> {
        if (indexes.length !== values.length) {
            GoPanic.raiseRuntime("array literal index/value length mismatch");
        }
        const result = GoArray.zero<T, N>(length, zero);
        for (let entry = 0; entry < indexes.length; entry++) {
            result.set(GoDenseIndex.get(indexes, entry), GoDenseIndex.get(values, entry));
        }
        return result;
    }
    public copy(): GoArray<T, N> {
        return new GoArray<T, N>(this.$values.slice(this.$offset, this.$offset + globalThis.Number(this.length)), 0, this.length);
    }
    public get(index: number | bigint): T {
        const offset: number = this.$check(index);
        return GoDenseIndex.get(this.$values, this.$offset + offset);
    }
    public set(index: number | bigint, value: T): void {
        const offset: number = this.$check(index);
        this.$values[this.$offset + offset] = value;
    }
    private $check(index: number | bigint): number {
        const offset: number = globalThis.Number(index);
        if (!globalThis.Number.isInteger(offset) || offset < 0 || offset >= this.length) {
            GoPanic.raiseRuntime("array index out of bounds");
        }
        return offset;
    }
    declare private readonly then?: never;
}
