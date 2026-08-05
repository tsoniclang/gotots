import type { Awaitable, bool } from "@gotots/gostdlib/internal/scalars.js";
type Yield<T> = (value: T) => Awaitable<bool>;
type Yield2<K, V> = (key: K, value: V) => Awaitable<bool>;

export class Seq<T> {
  constructor(
    readonly value: ((yieldValue: Yield<T> | undefined) => Awaitable<void>) | undefined,
  ) {}
}

export class Seq2<K, V> {
  constructor(
    readonly value: ((yieldValue: Yield2<K, V> | undefined) => Awaitable<void>) | undefined,
  ) {}
}
