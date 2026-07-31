import type { bool } from "@gotots/runtime/scalars.js";

export class Seq<
  T,
  Implementation = ((yieldValue: ((value: T) => bool) | undefined) => void) | undefined,
> {
  constructor(readonly value: Implementation) {}
}

export class Seq2<
  K,
  V,
  Implementation = ((yieldValue: ((key: K, value: V) => bool) | undefined) => void) | undefined,
> {
  constructor(readonly value: Implementation) {}
}
