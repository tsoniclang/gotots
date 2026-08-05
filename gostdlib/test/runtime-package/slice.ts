import { GoDenseIndex } from "./dense-index.js";
import { GoPanic } from "./panic.js";
export class RuntimeSlice<T> {
    protected constructor(private readonly backing: T[] | null, private readonly offset: number, readonly length: number, readonly capacity: number) {
    }
    static nil<T>(): RuntimeSlice<T> {
        return new RuntimeSlice<T>(null, 0, 0, 0);
    }
    static make<T>(length: number | bigint, capacity: (number | bigint) | null, zero: T): RuntimeSlice<T> {
        const numericLength = globalThis.Number(length);
        const resolvedCapacity = capacity === null ? numericLength : globalThis.Number(capacity);
        if (numericLength < 0 || resolvedCapacity < numericLength)
            GoPanic.raiseRuntime("slice bounds out of range");
        const backing = new Array<T>(resolvedCapacity).fill(zero);
        return new RuntimeSlice<T>(backing, 0, numericLength, resolvedCapacity);
    }
    static literal<T>(values: T[]): RuntimeSlice<T> {
        return new RuntimeSlice<T>(values, 0, values.length, values.length);
    }
    isNil(): boolean {
        return this.backing === null;
    }
    get(index: number | bigint): T {
        const numericIndex = globalThis.Number(index);
        const backing = this.backing;
        if (backing === null || (numericIndex < 0 || numericIndex >= this.length))
            GoPanic.raiseRuntime("runtime error: index out of range [" + String(numericIndex) + "] with length " + String(this.length));
        return GoDenseIndex.get(backing, this.offset + numericIndex);
    }
    set(index: number | bigint, value: T): T {
        const numericIndex = globalThis.Number(index);
        const backing = this.backing;
        if (backing === null || (numericIndex < 0 || numericIndex >= this.length))
            GoPanic.raiseRuntime("runtime error: index out of range [" + String(numericIndex) + "] with length " + String(this.length));
        backing[this.offset + numericIndex] = value;
        return value;
    }
    slice(low: number | bigint, high: (number | bigint) | null, max: (number | bigint) | null): RuntimeSlice<T> {
        const numericLow = globalThis.Number(low);
        const resolvedHigh = high === null ? this.length : globalThis.Number(high);
        const resolvedMax = max === null ? this.capacity : globalThis.Number(max);
        if (numericLow < 0 || resolvedHigh < numericLow || (resolvedMax < resolvedHigh || resolvedMax > this.capacity))
            GoPanic.raiseRuntime("slice bounds out of range");
        return new RuntimeSlice<T>(this.backing, this.offset + numericLow, resolvedHigh - numericLow, resolvedMax - numericLow);
    }
    append(zero: T, values: T[]): RuntimeSlice<T> {
        const newLength = this.length + values.length;
        const existingBacking = this.backing;
        if (values.length === 0)
            return this;
        if (newLength <= this.capacity) {
            if (existingBacking === null)
                GoPanic.raiseRuntime("slice bounds out of range");
            for (let index = 0; index < values.length; index++) {
                existingBacking[this.offset + this.length + index] = GoDenseIndex.get(values, index);
            }
            return new RuntimeSlice<T>(existingBacking, this.offset, newLength, this.capacity);
        }
        let nextCapacity = this.capacity === 0 ? 1 : this.capacity * 2;
        while (nextCapacity < newLength) {
            nextCapacity = nextCapacity * 2;
        }
        const backing = new Array<T>(nextCapacity).fill(zero);
        if (existingBacking !== null) {
            for (let index = 0; index < this.length; index++) {
                backing[index] = GoDenseIndex.get(existingBacking, this.offset + index);
            }
        }
        for (let index = 0; index < values.length; index++) {
            backing[this.length + index] = GoDenseIndex.get(values, index);
        }
        return new RuntimeSlice<T>(backing, 0, newLength, nextCapacity);
    }
    static copy<T>(target: RuntimeSlice<T>, source: RuntimeSlice<T>): number {
        const count = Math.min(target.length, source.length);
        const targetBacking = target.backing;
        const sourceBacking = source.backing;
        if (count === 0)
            return 0;
        if (targetBacking !== null && sourceBacking !== null) {
            if (targetBacking === sourceBacking)
                targetBacking.copyWithin(target.offset, source.offset, source.offset + count);
            else
                for (let index = 0; index < count; index++) {
                    targetBacking[target.offset + index] = GoDenseIndex.get(sourceBacking, source.offset + index);
                }
            return count;
        }
        const values = new Array<T>(count);
        for (let index = 0; index < count; index++) {
            values[index] = source.get(index);
        }
        for (let index = 0; index < count; index++) {
            target.set(index, GoDenseIndex.get(values, index));
        }
        return count;
    }
}
