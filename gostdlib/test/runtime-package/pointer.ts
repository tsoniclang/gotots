import { GoDenseIndex } from "./dense-index.js";
import { GoPanic } from "./panic.js";
export class GoPointer<L, S> {
    declare private type: (v: L) => L;
    private static readonly roots: WeakMap<object, object> = new WeakMap<object, object>;
    private static readonly children: WeakMap<object, Map<PropertyKey | bigint, object>> = new WeakMap<object, Map<PropertyKey | bigint, object>>;
    private $go$resolvedAddress?: object;
    private constructor(private readonly $go$getAddress: () => object, private read: () => S, private write: (value: S) => void, private readonly $go$region?: readonly [
        S[],
        number
    ]) {
    }
    get $go$address(): object {
        return this.$go$resolvedAddress ??= this.$go$getAddress();
    }
    private static root(owner: object): object {
        let address = GoPointer.roots.get(owner);
        if (address === void 0) {
            address = {};
            GoPointer.roots.set(owner, address);
        }
        return address;
    }
    static child(parent: object, key: PropertyKey | bigint): object {
        let children = GoPointer.children.get(parent);
        if (children === void 0) {
            children = new Map<PropertyKey | bigint, object>;
            GoPointer.children.set(parent, children);
        }
        let address = children.get(key);
        if (address === void 0) {
            address = {};
            children.set(key, address);
        }
        return address;
    }
    static cell<L, S>(value: S): GoPointer<L, S> {
        const storage: [
            S
        ] = [value];
        return new GoPointer<L, S>(() => storage, () => storage[0], (next: S) => storage[0] = next, [storage, 0]);
    }
    static field<L, PL, PS extends object, K extends keyof PS>(parent: GoPointer<PL, PS>, key: K): GoPointer<L, PS[K]> {
        return new GoPointer<L, PS[K]>(() => GoPointer.child(parent.$go$address, key), () => parent.read()[key], (next: PS[K]) => parent.read()[key] = next);
    }
    static objectField<L, O extends object, K extends keyof O>(owner: O, key: K): GoPointer<L, O[K]> {
        return new GoPointer<L, O[K]>(() => GoPointer.child(GoPointer.root(owner), key), () => owner[key], (next: O[K]) => owner[key] = next);
    }
    static element<L, S>(location: readonly [
        S[],
        number
    ]): GoPointer<L, S> {
        const backing = location[0];
        const index = location[1];
        return new GoPointer<L, S>(() => GoPointer.child(GoPointer.root(backing), index), () => GoDenseIndex.get(backing, index), (next: S) => backing[index] = next, location);
    }
    static index<L, S, PL, O extends {
        get(index: number | bigint): S;
        set(index: number | bigint, value: S): void;
    }>(parent: GoPointer<PL, O>, index: number | bigint): GoPointer<L, S> {
        const selected = GoPointer.dereference(parent);
        const numericIndex = globalThis.Number(index);
        selected.read().get(numericIndex);
        return new GoPointer<L, S>(() => GoPointer.child(selected.$go$address, numericIndex), () => selected.read().get(numericIndex), (next: S) => selected.read().set(numericIndex, next));
    }
    static arrayRegion<L, T, S extends {
        length: number;
        get(index: number | bigint): T;
        set(index: number | bigint, value: T): void;
    }>(location: readonly [
        T[],
        number
    ], view: S): GoPointer<L, S> {
        const backing = location[0];
        const offset = location[1];
        return new GoPointer<L, S>(() => GoPointer.child(GoPointer.root(backing), offset), () => view, (next: S) => {
            for (let index = 0; index < view.length; index++) {
                view.set(index, next.get(index));
            }
        });
    }
    static equal<LL, LS, RL, RS>(left: GoPointer<LL, LS> | undefined, right: GoPointer<RL, RS> | undefined): boolean {
        return left === right || left?.$go$address === right?.$go$address;
    }
    static dereference<L, S>(pointer: GoPointer<L, S> | undefined): GoPointer<L, S> {
        if (pointer === void 0) {
            GoPanic.raiseRuntime("nil pointer dereference");
        }
        return pointer;
    }
    static direct<L>(pointer: L | undefined): L {
        if (pointer === void 0) {
            GoPanic.raiseRuntime("nil pointer dereference");
        }
        return pointer;
    }
    static view<F, T, S>(pointer: GoPointer<F, S> | undefined): GoPointer<T, S> | undefined {
        if (pointer === void 0) {
            return void 0;
        }
        return new GoPointer<T, S>(() => pointer.$go$address, pointer.read, pointer.write, pointer.$go$region);
    }
    get value(): S {
        return this.read();
    }
    set value(value: S) {
        this.write(value);
    }
    static region<L, S>(pointer: GoPointer<L, S> | undefined, length: number | bigint): readonly [
        S[],
        number
    ] | undefined {
        const requested = globalThis.Number(length);
        if (requested < 0)
            GoPanic.raiseRuntime("unsafe length is negative");
        if (pointer === void 0) {
            if (requested === 0) {
                return void 0;
            }
            GoPanic.raiseRuntime("unsafe operation on nil pointer");
        }
        const region = pointer.$go$region;
        if (region === void 0 || requested > region[0].length - region[1])
            GoPanic.raiseRuntime("unsafe operation requires a contiguous pointer region");
        return region;
    }
    private $go$rawAccess?: readonly [
        () => S,
        (value: S) => void
    ];
    static $go$unsafeBind<L, S>(pointer: GoPointer<L, S>, before: () => void, after: () => void): void {
        const raw = pointer.$go$rawAccess ??= [pointer.read, pointer.write];
        pointer.read = () => {
            before();
            return raw[0]();
        };
        pointer.write = (value: S) => {
            before();
            raw[1](value);
            after();
        };
    }
    static $go$unsafeMemory<L, S>(pointer: GoPointer<L, S>): readonly [
        object,
        () => S,
        (value: S) => void,
        readonly [
            S[],
            number
        ] | undefined
    ] {
        const raw = pointer.$go$rawAccess;
        return [pointer.$go$address, () => (raw === void 0 ? pointer.read : raw[0])(), (next: S) => (raw === void 0 ? pointer.write : raw[1])(next), pointer.$go$region];
    }
    static $go$unsafeView<L, S>(address: object, read: () => S, write: (value: S) => void, region: readonly [
        S[],
        number
    ] | undefined): GoPointer<L, S> {
        return new GoPointer<L, S>(() => address, read, write, region);
    }
}
export function goPointerRegion<L, S>(pointer: GoPointer<L, S> | undefined, length: number | bigint): readonly [
    S[],
    number
] | undefined {
    return GoPointer.region<L, S>(pointer, length);
}
export function goPointerUnsafeMemory<L, S>(pointer: GoPointer<L, S>): readonly [
    object,
    () => S,
    (value: S) => void,
    readonly [
        S[],
        number
    ] | undefined
] {
    return GoPointer.$go$unsafeMemory<L, S>(pointer);
}
