export declare class GoArray<T, N extends number> {
  readonly length: N;
  static zero<T, N extends number>(length: N, zero: T): GoArray<T, N>;
  static literal<T, N extends number>(length: N, zero: T, indexes: number[], values: T[]): GoArray<T, N>;
  copy(): GoArray<T, N>;
  get(index: number | bigint): T;
  set(index: number | bigint, value: T): void;
}
