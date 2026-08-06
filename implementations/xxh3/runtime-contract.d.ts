declare module "@gotots/runtime/scalars.js" {
  export type bool = boolean;
  export type gostring = string;
  export type int64 = number;
  export type uint8 = number;
  export type uint64 = number;
}

declare module "@gotots/runtime/array.js" {
  export class GoArray<T, N extends number> {
    static literal<T, N extends number>(
      length: N,
      zero: T,
      indexes: number[],
      values: T[],
    ): GoArray<T, N>;
    get(index: number): T;
  }
}

declare module "@gotots/runtime/pointer.js" {
  export class GoPointer<L, S = L> {
    static direct<L>(pointer: L | undefined): L;
  }
}

declare module "@gotots/runtime/slice.js" {
  export class RuntimeSlice<T> {
    readonly length: number;
    get(index: number): T;
  }
}

declare module "@gotots/runtime/interface-value.js" {
  export interface GoError {
    Error(): string | Promise<string>;
  }
}

declare module "@gotots/runtime/unsafe-pointer.js" {
  export class GoUnsafePointer {}
}
