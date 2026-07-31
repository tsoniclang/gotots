import { GoPanic } from "./panic.js";
export class GoPointer<L, S> {
    declare private readonly logical: (value: L) => L;
    private static readonly roots: WeakMap<object, object> = new WeakMap<object, object>;
    private static readonly children: WeakMap<object, Map<PropertyKey | bigint, object>> = new WeakMap<object, Map<PropertyKey | bigint, object>>;
    private constructor(readonly $go$address: object, private readonly read: () => S, private readonly write: (value: S) => void) {
    }
    private static root(owner: object): object {
        let address = GoPointer.roots.get(owner);
        if (address === void 0) {
            address = {};
            GoPointer.roots.set(owner, address);
        }
        return address;
    }
    private static child(parent: object, key: PropertyKey | bigint): object {
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
        return new GoPointer<L, S>(storage, () => {
            return storage[0];
        }, (next: S) => {
            storage[0] = next;
        });
    }
    static field<L, PL, PS extends object, K extends keyof PS>(parent: GoPointer<PL, PS>, key: K): GoPointer<L, PS[K]> {
        return new GoPointer<L, PS[K]>(GoPointer.child(parent.$go$address, key), () => {
            return parent.value[key];
        }, (next: PS[K]) => {
            parent.value[key] = next;
        });
    }
    static objectField<L, O extends object, K extends keyof O>(owner: O, key: K): GoPointer<L, O[K]> {
        return new GoPointer<L, O[K]>(GoPointer.child(GoPointer.root(owner), key), () => {
            return owner[key];
        }, (next: O[K]) => {
            owner[key] = next;
        });
    }
    static element<L, S>(location: readonly [
        S[],
        number
    ]): GoPointer<L, S> {
        const backing = location[0];
        const index = location[1];
        return new GoPointer<L, S>(GoPointer.child(GoPointer.root(backing), index), () => {
            return backing[index]!;
        }, (next: S) => {
            backing[index] = next;
        });
    }
    static index<L, S, PL, O extends {
        get(index: number | bigint): S;
        set(index: number | bigint, value: S): void;
    }>(parent: GoPointer<PL, O>, index: number | bigint): GoPointer<L, S> {
        const selected = GoPointer.dereference(parent);
        const numericIndex = globalThis.Number(index);
        selected.value.get(numericIndex);
        return new GoPointer<L, S>(GoPointer.child(selected.$go$address, numericIndex), () => {
            return selected.value.get(numericIndex);
        }, (next: S) => {
            selected.value.set(numericIndex, next);
        });
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
        return new GoPointer<L, S>(GoPointer.child(GoPointer.root(backing), offset), () => {
            return view;
        }, (next: S) => {
            for (let index = 0; index < view.length; index++) {
                view.set(index, next.get(index));
            }
        });
    }
    static equal<LL, LS, RL, RS>(left: GoPointer<LL, LS> | undefined, right: GoPointer<RL, RS> | undefined): boolean {
        return left === right || left !== void 0 && right !== void 0 && left.$go$address === right.$go$address;
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
        return new GoPointer<T, S>(pointer.$go$address, pointer.read, pointer.write);
    }
    get value(): S {
        return this.read();
    }
    set value(value: S) {
        this.write(value);
    }
}
