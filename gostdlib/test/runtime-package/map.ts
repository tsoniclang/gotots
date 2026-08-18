import { GoPanic } from "./panic.js";
export abstract class GoMapValue<K, V> {
    abstract lookup(key: K): V;
    abstract lookupOk(key: K): [
        V,
        boolean
    ];
    abstract store(key: K, value: V): void;
    abstract delete(key: K): void;
    abstract length(): number;
    abstract isNil(): boolean;
    abstract clear(): void;
    abstract keys(): K[];
    declare private readonly then?: never;
}
export class GoMap<K extends boolean | number | bigint | string, V> extends GoMapValue<K, V> {
    private constructor(private readonly zeroValue: V, private readonly values: Map<K, V> | undefined) {
        super();
    }
    static nil<K extends boolean | number | bigint | string, V>(zeroValue: V): GoMap<K, V> {
        return new GoMap<K, V>(zeroValue, undefined);
    }
    static make<K extends boolean | number | bigint | string, V>(zeroValue: V, size: number | bigint, entries: [
        K,
        V
    ][]): GoMap<K, V> {
        return new GoMap<K, V>(zeroValue, new Map<K, V>(entries));
    }
    lookup(key: K): V {
        const storage = this.values;
        if (storage === undefined) {
            return this.zeroValue;
        }
        const storedValue = storage.get(key);
        if (storedValue === undefined) {
            return this.zeroValue;
        }
        return storedValue;
    }
    lookupOk(key: K): [
        V,
        boolean
    ] {
        const storage = this.values;
        if (storage === undefined) {
            return [this.zeroValue, false];
        }
        const storedValue = storage.get(key);
        if (storedValue === undefined) {
            return [this.zeroValue, false];
        }
        return [storedValue, true];
    }
    store(key: K, value: V): void {
        if (this.values === undefined) {
            GoPanic.raiseRuntime("assignment to entry in nil map");
        }
        this.values.set(key, value);
    }
    delete(key: K): void {
        if (this.values !== undefined) {
            this.values.delete(key);
        }
    }
    length(): number {
        return this.values !== undefined ? this.values.size : 0;
    }
    isNil(): boolean {
        return this.values === undefined;
    }
    clear(): void {
        if (this.values !== undefined) {
            this.values.clear();
        }
    }
    keys(): K[] {
        return this.values !== undefined ? Array.from(this.values.keys()) : [];
    }
}
export class GoMapHash {
    private static readonly objects: WeakMap<object, number> = new WeakMap<object, number>;
    private static nextObject: number = 1;
    static boolean(value: boolean): number {
        return value ? 1 : 0;
    }
    static number(value: number): number {
        return Math.trunc(value) >>> 0;
    }
    static bigint(value: bigint): number {
        return globalThis.Number(BigInt.asUintN(32, value));
    }
    static string(value: string): number {
        let hash = 2166136261;
        for (let index = 0; index < value.length; index++) {
            hash = GoMapHash.mix(hash, value.charCodeAt(index));
        }
        return hash;
    }
    static object(value: object): number {
        let result = GoMapHash.objects.get(value);
        if (result === undefined) {
            result = GoMapHash.nextObject;
            GoMapHash.nextObject++;
            GoMapHash.objects.set(value, result);
        }
        return result;
    }
    static mix(hash: number, next: number): number {
        return Math.imul(hash ^ next, 16777619) >>> 0;
    }
    declare private readonly then?: never;
}
