import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoMapValue } from "@gotots/runtime/map.js";
import type { bool } from "@gotots/runtime/scalars.js";

import { Seq } from "../iter/sequence.js";

type PrimitiveComparable = boolean | number | bigint | string;
type Equality<L, R> = ((left: L, right: R) => bool) | undefined;

export function Copy<K, V>(
  target: GoMapValue<K, V>,
  source: GoMapValue<K, V>,
): void {
  for (const key of source.keys()) {
    target.store(key, source.lookup(key));
  }
}

export function Equal<K, V extends PrimitiveComparable>(
  left: GoMapValue<K, V>,
  right: GoMapValue<K, V>,
): bool {
  if (left.length() !== right.length()) {
    return false;
  }
  for (const key of left.keys()) {
    const [rightValue, present] = right.lookupOk(key);
    if (!present || left.lookup(key) !== rightValue) {
      return false;
    }
  }
  return true;
}

export function EqualFunc<K, L, R>(
  left: GoMapValue<K, L>,
  right: GoMapValue<K, R>,
  equal: Equality<L, R>,
): bool {
  if (left.length() !== right.length()) {
    return false;
  }
  for (const key of left.keys()) {
    const [rightValue, present] = right.lookupOk(key);
    if (!present || !callEquality(equal, left.lookup(key), rightValue)) {
      return false;
    }
  }
  return true;
}

export function Keys<K, V>(source: GoMapValue<K, V>): Seq<K> {
  return new Seq<K>((yieldValue): void => {
    if (yieldValue === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    for (const key of source.keys()) {
      if (!yieldValue(key)) {
        return;
      }
    }
  });
}

export function Values<K, V>(source: GoMapValue<K, V>): Seq<V> {
  return new Seq<V>((yieldValue): void => {
    if (yieldValue === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    for (const key of source.keys()) {
      if (!yieldValue(source.lookup(key))) {
        return;
      }
    }
  });
}

function callEquality<L, R>(
  equal: Equality<L, R>,
  left: L,
  right: R,
): bool {
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return equal(left, right);
}
