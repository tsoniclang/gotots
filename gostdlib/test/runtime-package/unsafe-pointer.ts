import { GoDenseIndex } from "./dense-index.js";
import { GoPanic } from "./panic.js";
import { GoPointer, goPointerUnsafeMemory } from "./pointer.js";
export class GoUnsafeCodec<S> {
    private readonly bindings: WeakMap<object, readonly [
        object,
        () => S,
        (value: S) => void,
        readonly [
            S[],
            number
        ] | undefined
    ]> = new WeakMap;
    constructor(public readonly size: number, private readonly readValue: (bytes: Uint8Array, offset: number) => S, private readonly writeValue: (value: S, bytes: Uint8Array, offset: number) => void) {
        if (!globalThis.Number.isSafeInteger(size) || size < 0)
            GoPanic.raiseRuntime("unsafe codec size is invalid");
    }
    public read(bytes: Uint8Array, offset: number): S {
        if (!globalThis.Number.isSafeInteger(offset) || (offset < 0 || offset + this.size > bytes.length))
            GoPanic.raiseRuntime("unsafe codec access is out of range");
        return this.readValue(bytes, offset);
    }
    public write(value: S, bytes: Uint8Array, offset: number): void {
        if (!globalThis.Number.isSafeInteger(offset) || (offset < 0 || offset + this.size > bytes.length))
            GoPanic.raiseRuntime("unsafe codec access is out of range");
        this.writeValue(value, bytes, offset);
    }
    public decode(bytes: Uint8Array): S {
        return this.read(bytes, 0);
    }
    public encode(value: S): Uint8Array {
        const bytes: Uint8Array = new Uint8Array(this.size);
        this.write(value, bytes, 0);
        return bytes;
    }
    public $bind(pointer: object, memory: readonly [
        object,
        () => S,
        (value: S) => void,
        readonly [
            S[],
            number
        ] | undefined
    ]): void {
        this.bindings.set(pointer, memory);
    }
    public $bound(pointer: object): readonly [
        object,
        () => S,
        (value: S) => void,
        readonly [
            S[],
            number
        ] | undefined
    ] | undefined {
        return this.bindings.get(pointer);
    }
}
export class GoUnsafePointer {
    private static nextBase: number = 4096;
    private static readonly roots: WeakMap<object, GoUnsafePointer> = new WeakMap;
    private static readonly locations: WeakMap<object, GoUnsafePointer> = new WeakMap;
    private static readonly allocations: GoUnsafePointer[] = [];
    private constructor(private readonly base: number, private readonly offset: number, private readonly length: number, private readonly readBytes: (offset: number, length: number) => Uint8Array, private readonly writeBytes: (offset: number, bytes: Uint8Array) => void, private readonly children: Map<number, GoUnsafePointer>, private readonly flushes: (() => void)[], private readonly refreshes: (() => void)[]) {
        GoUnsafePointer.locations.set(this, this);
    }
    private get address(): number {
        return this.base + this.offset;
    }
    private flush(): void {
        for (const callback of this.flushes) {
            callback();
        }
    }
    private refresh(): void {
        for (const callback of this.refreshes) {
            callback();
        }
    }
    private at(offset: number): GoUnsafePointer {
        if (!globalThis.Number.isSafeInteger(offset) || (offset < 0 || offset > this.length))
            GoPanic.raiseRuntime("unsafe pointer offset is outside its allocation");
        let existing: GoUnsafePointer | undefined = this.children.get(offset);
        if (existing !== void 0) {
            return existing;
        }
        existing = new GoUnsafePointer(this.base, offset, this.length, this.readBytes, this.writeBytes, this.children, this.flushes, this.refreshes);
        this.children.set(offset, existing);
        return existing;
    }
    public static from<L, S>(value: GoPointer<L, S> | undefined, codec: GoUnsafeCodec<S>): GoUnsafePointer | undefined {
        if (value === void 0) {
            return void 0;
        }
        const memory = goPointerUnsafeMemory<L, S>(value);
        const located = GoUnsafePointer.locations.get(memory[0]);
        if (located !== void 0) {
            codec.$bind(located, memory);
            GoPointer.$go$unsafeBind(value, (): void => located.flush(), (): void => located.refresh());
            return located;
        }
        const region = memory[3];
        const rootKey: object = region === void 0 ? memory[0] : region[0];
        let root: GoUnsafePointer | undefined = GoUnsafePointer.roots.get(rootKey);
        if (root === void 0) {
            let totalLength: number;
            let readBytes: (offset: number, length: number) => Uint8Array;
            let writeBytes: (offset: number, bytes: Uint8Array) => void;
            if (region === void 0) {
                totalLength = codec.size;
                readBytes = (offset: number, length: number) => {
                    const bytes: Uint8Array = codec.encode(memory[1]());
                    return bytes.slice(offset, offset + length);
                };
                writeBytes = (offset: number, replacement: Uint8Array) => {
                    const bytes: Uint8Array = codec.encode(memory[1]());
                    bytes.set(replacement, offset);
                    memory[2](codec.decode(bytes));
                };
            }
            else {
                const backing = region[0];
                totalLength = backing.length * codec.size;
                readBytes = (offset: number, length: number) => {
                    const first: number = Math.floor(offset / codec.size);
                    const last: number = Math.ceil((offset + length) / codec.size);
                    const bytes: Uint8Array = new Uint8Array((last - first) * codec.size);
                    for (let index = first; index < last; index++) {
                        codec.write(GoDenseIndex.get(backing, index), bytes, (index - first) * codec.size);
                    }
                    const start: number = offset - first * codec.size;
                    return bytes.slice(start, start + length);
                };
                writeBytes = (offset: number, replacement: Uint8Array) => {
                    const first: number = Math.floor(offset / codec.size);
                    const last: number = Math.ceil((offset + replacement.length) / codec.size);
                    const bytes: Uint8Array = new Uint8Array((last - first) * codec.size);
                    for (let index = first; index < last; index++) {
                        codec.write(GoDenseIndex.get(backing, index), bytes, (index - first) * codec.size);
                    }
                    const start: number = offset - first * codec.size;
                    bytes.set(replacement, start);
                    for (let index = first; index < last; index++) {
                        backing[index] = codec.read(bytes, (index - first) * codec.size);
                    }
                };
            }
            root = new GoUnsafePointer(GoUnsafePointer.nextBase, 0, totalLength, readBytes, writeBytes, new Map, [], []);
            GoUnsafePointer.nextBase = GoUnsafePointer.nextBase + totalLength + 4096;
            GoUnsafePointer.roots.set(rootKey, root);
            GoUnsafePointer.allocations.push(root);
        }
        const result: GoUnsafePointer = root.at(region === void 0 ? 0 : region[1] * codec.size);
        codec.$bind(result, memory);
        GoPointer.$go$unsafeBind(value, (): void => result.flush(), (): void => result.refresh());
        return result;
    }
    public static to<L, S>(value: GoUnsafePointer | undefined, codec: GoUnsafeCodec<S>): GoPointer<L, S> | undefined {
        if (value === void 0) {
            return void 0;
        }
        const bound = codec.$bound(value);
        if (bound !== void 0) {
            return GoPointer.$go$unsafeView<L, S>(bound[0], bound[1], bound[2], bound[3]);
        }
        let current: S = codec.read(value.readBytes(value.offset, codec.size), 0);
        const read = (): S => current;
        const write = (next: S): void => {
            current = next;
            value.writeBytes(value.offset, codec.encode(current));
            value.refresh();
        };
        const flush = (): void => value.writeBytes(value.offset, codec.encode(current));
        const refresh = (): void => {
            current = codec.read(value.readBytes(value.offset, codec.size), 0);
        };
        value.flushes.push(flush);
        value.refreshes.push(refresh);
        codec.$bind(value, [value, read, write, void 0]);
        return GoPointer.$go$unsafeView<L, S>(value, read, write, void 0);
    }
    private static allocationAtAddress(value: number): GoUnsafePointer | undefined {
        let low: number = 0;
        let high: number = GoUnsafePointer.allocations.length - 1;
        while (low <= high) {
            const middle: number = Math.floor((low + high) / 2);
            const candidate: GoUnsafePointer = GoDenseIndex.get(GoUnsafePointer.allocations, middle);
            if (candidate.base <= value) {
                low = middle + 1;
            }
            else {
                high = middle - 1;
            }
        }
        if (high < 0) {
            return void 0;
        }
        const candidate: GoUnsafePointer = GoDenseIndex.get(GoUnsafePointer.allocations, high);
        if (value <= candidate.base + candidate.length) {
            return candidate;
        }
        return void 0;
    }
    public static fromRelative(value: GoUnsafePointer | undefined, address: number | bigint, zero: number | bigint): GoUnsafePointer | undefined {
        if (value === void 0) {
            if (address === zero) {
                return void 0;
            }
            else
                GoPanic.raiseRuntime("unsafe integer address does not identify live generated memory");
        }
        const numeric: number = globalThis.Number(address);
        if (!globalThis.Number.isSafeInteger(numeric))
            GoPanic.raiseRuntime("unsafe integer address is not representable");
        const offset: number = numeric - value.base;
        return value.at(offset);
    }
    public static fromInteger(value: number | bigint, zero: number | bigint): GoUnsafePointer | undefined {
        if (value === zero) {
            return void 0;
        }
        const numeric: number = globalThis.Number(value);
        if (!globalThis.Number.isSafeInteger(numeric))
            GoPanic.raiseRuntime("unsafe integer address is not representable");
        const allocation: GoUnsafePointer | undefined = GoUnsafePointer.allocationAtAddress(numeric);
        if (allocation === void 0)
            GoPanic.raiseRuntime("unsafe integer address does not identify live generated memory");
        const offset: number = numeric - allocation.base;
        return allocation.at(offset);
    }
    public static toInteger(value: GoUnsafePointer | undefined, zero: number): number;
    public static toInteger(value: GoUnsafePointer | undefined, zero: bigint): bigint;
    public static toInteger(value: GoUnsafePointer | undefined, zero: number | bigint): number | bigint {
        if (value === void 0) {
            return zero;
        }
        return typeof zero === "bigint" ? globalThis.BigInt(value.address) : value.address;
    }
}
