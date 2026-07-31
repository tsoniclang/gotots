export declare class RuntimeSlice<T> {
  readonly length: number;
  readonly capacity: number;

  static nil<T>(): RuntimeSlice<T>;
  static make<T>(length: number | bigint, capacity: number | bigint | null, zero: T): RuntimeSlice<T>;
  static literal<T>(values: T[]): RuntimeSlice<T>;
  static copy<T>(target: RuntimeSlice<T>, source: RuntimeSlice<T>): number;

  isNil(): boolean;
  get(index: number | bigint): T;
  set(index: number | bigint, value: T): T;
  slice(low: number | bigint, high: number | bigint | null, max: number | bigint | null): RuntimeSlice<T>;
  append(zero: T, values: T[]): RuntimeSlice<T>;
  appendSlice(zero: T, source: RuntimeSlice<T>): RuntimeSlice<T>;
  clear(zero: T): void;
  toArray(): T[];
}
