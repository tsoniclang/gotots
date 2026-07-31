export declare class RuntimeSlice<T> {
  readonly length: number;
  readonly capacity: number;

  static nil<T>(): RuntimeSlice<T>;
  static literal<T>(values: T[]): RuntimeSlice<T>;

  isNil(): boolean;
  get(index: number | bigint): T;
  set(index: number | bigint, value: T): T;
  toArray(): T[];
}
