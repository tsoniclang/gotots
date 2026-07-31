import { GoPanic } from "./panic.js";
export class GoMap<K extends boolean | number | bigint | string, V> {
    private constructor(private readonly zeroValue: V, private readonly values: Map<K, V> | undefined) {
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
export interface GoMapValue<K, V> {
    lookup(key: K): V;
    lookupOk(key: K): [
        V,
        boolean
    ];
    store(key: K, value: V): void;
    delete(key: K): void;
    length(): number;
    isNil(): boolean;
    clear(): void;
    keys(): K[];
}
