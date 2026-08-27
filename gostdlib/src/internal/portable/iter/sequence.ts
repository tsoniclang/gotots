import type { bool } from "@gotots/gostdlib/internal/scalars.js";
type Yield<T> = (value: T) => bool;
type Yield2<K, V> = (key: K, value: V) => bool;

export class Seq<T> {
  constructor(
    readonly value: ((yieldValue: Yield<T> | undefined) => void) | undefined,
  ) {}
}

export class Seq2<K, V> {
  constructor(
    readonly value: ((yieldValue: Yield2<K, V> | undefined) => void) | undefined,
  ) {}
}
