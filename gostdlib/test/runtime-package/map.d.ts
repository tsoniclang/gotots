export interface GoMapValue<K, V> {
  lookup(key: K): V;
  lookupOk(key: K): [V, boolean];
  store(key: K, value: V): void;
  delete(key: K): void;
  length(): number;
  isNil(): boolean;
  clear(): void;
  keys(): K[];
}

export declare class GoMap<K extends boolean | number | bigint | string, V> implements GoMapValue<K, V> {
  static nil<K extends boolean | number | bigint | string, V>(zeroValue: V): GoMap<K, V>;
  static make<K extends boolean | number | bigint | string, V>(zeroValue: V, size: number | bigint, entries: [K, V][]): GoMap<K, V>;
  lookup(key: K): V;
  lookupOk(key: K): [V, boolean];
  store(key: K, value: V): void;
  delete(key: K): void;
  length(): number;
  isNil(): boolean;
  clear(): void;
  keys(): K[];
}
